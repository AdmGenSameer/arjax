"""ML-based smart package recommendations using TF-IDF and cosine similarity.

Provides content-based software discovery, alternative tool matching,
and semantic query understanding. Works with scikit-learn when installed,
with a zero-dependency pure-Python fallback.
"""

import math
import re
from collections import Counter
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, List, Optional, Set

from arjax.config.logging import get_logger
from arjax.installation.recipes import RecipeStore

logger = get_logger(__name__)

# Check scikit-learn availability
try:
    from sklearn.feature_extraction.text import TfidfVectorizer
    from sklearn.metrics.pairwise import cosine_similarity
    SKLEARN_AVAILABLE = True
except ImportError:
    SKLEARN_AVAILABLE = False


@dataclass
class RecommendationResult:
    """Represents a recommended package match."""
    name: str
    description: str
    provider: str
    similarity_score: float

    @property
    def match_percentage(self) -> int:
        """Return similarity score as a display percentage."""
        return max(1, min(100, int(round(self.similarity_score * 100))))


class MLPackageSuggester:
    """Smart package recommender using TF-IDF and cosine similarity."""

    DEFAULT_CACHE_DIR = Path.home() / ".cache" / "arjax"
    CACHE_FILE = DEFAULT_CACHE_DIR / "ml_tfidf_cache.pkl"
    DEFAULT_STOP_WORDS = {
        "a", "an", "the", "and", "or", "in", "on", "at", "to", "for", "of",
        "with", "is", "it", "by", "from", "as", "into", "like", "through",
        "after", "over", "between", "out", "against", "during", "without",
        "before", "under", "around", "among", "built", "fast", "native"
    }

    # Curated knowledge graph for proprietary/popular tool alternatives
    POPULAR_ALTERNATIVES: Dict[str, List[Dict[str, str]]] = {
        "photoshop": [
            {"name": "gimp", "desc": "GNU Image Manipulation Program",
             "prov": "repo"},
            {"name": "krita", "desc": "Digital painting & sketching tool",
             "prov": "repo"},
            {"name": "inkscape", "desc": "Vector graphics editor",
             "prov": "repo"},
        ],
        "illustrator": [
            {"name": "inkscape", "desc": "Vector graphics editor",
             "prov": "repo"},
        ],
        "premiere": [
            {"name": "kdenlive", "desc": "Non-linear video editor",
             "prov": "repo"},
            {"name": "shotcut", "desc": "Cross-platform video editor",
             "prov": "repo"},
            {"name": "davinci-resolve", "desc": "Professional video editor",
             "prov": "aur"},
        ],
        "after-effects": [
            {"name": "natron", "desc": "Node-based compositing software",
             "prov": "repo"},
        ],
        "office": [
            {"name": "libreoffice", "desc": "Full-featured office suite",
             "prov": "repo"},
            {"name": "onlyoffice-bin", "desc": "Office productivity suite",
             "prov": "repo"},
        ],
        "word": [
            {"name": "libreoffice-writer", "desc": "Word processor",
             "prov": "repo"},
            {"name": "abiword", "desc": "Lightweight word processor",
             "prov": "repo"},
        ],
        "excel": [
            {"name": "libreoffice-calc", "desc": "Spreadsheet program",
             "prov": "repo"},
            {"name": "gnumeric", "desc": "Lightweight spreadsheet app",
             "prov": "repo"},
        ],
        "powerpoint": [
            {"name": "libreoffice-impress", "desc": "Presentation editor",
             "prov": "repo"},
        ],
        "notepad++": [
            {"name": "notepadqq", "desc": "Notepad++-like editor for Linux",
             "prov": "repo"},
            {"name": "kate", "desc": "Advanced KDE text editor",
             "prov": "repo"},
            {"name": "geany", "desc": "Fast & lightweight IDE & editor",
             "prov": "repo"},
        ],
        "whatsapp": [
            {"name": "zapfast", "desc": "Native WhatsApp client in Rust",
             "prov": "recipe"},
            {"name": "whatsapp-for-linux", "desc": "WhatsApp desktop client",
             "prov": "flatpak"},
        ],
        "slack": [
            {"name": "slack-desktop", "desc": "Slack communication app",
             "prov": "flatpak"},
            {"name": "mattermost-desktop",
             "desc": "Open-source Slack alternative", "prov": "repo"},
        ],
        "telegram": [
            {"name": "telegram-desktop", "desc": "Telegram messaging client",
             "prov": "repo"},
        ],
        "discord": [
            {"name": "discord", "desc": "All-in-one voice and text chat",
             "prov": "repo"},
            {"name": "vesktop",
             "desc": "Discord client with performance tweaks",
             "prov": "flatpak"},
        ],
        "rufus": [
            {"name": "balena-etcher", "desc": "Flash OS images to USB drives",
             "prov": "appimage"},
            {"name": "ventoy-bin", "desc": "New bootable USB solution",
             "prov": "aur"},
        ],
        "7zip": [
            {"name": "p7zip", "desc": "Command-line 7-Zip file archiver",
             "prov": "repo"},
            {"name": "peazip", "desc": "Archiving utility for 200+ formats",
             "prov": "repo"},
        ],
    }

    def __init__(
        self,
        top_n: int = 5,
        min_score: float = 0.10,
        force_pure_python: bool = False,
    ):
        self.top_n = top_n
        self.min_score = min_score
        self.force_pure_python = force_pure_python
        self.use_sklearn = SKLEARN_AVAILABLE and not force_pure_python

        self.packages: List[Dict[str, str]] = []
        self.vectorizer: Optional[Any] = None
        self.tfidf_matrix: Optional[Any] = None

        # Pure-python structures
        self._doc_frequencies: Dict[str, int] = {}
        self._idf: Dict[str, float] = {}
        self._doc_vectors: List[Dict[str, float]] = []

        self._build_corpus()
        self._fit_engine()

    def _build_corpus(self) -> None:
        """Harvest package metadata from recipes and mappings."""
        seen_names: Set[str] = set()
        corpus: List[Dict[str, str]] = []

        # 1. Load Declarative Recipes
        try:
            store = RecipeStore()
            for loc in store.load_all():
                recipe = loc.recipe
                name = recipe.name.lower().strip()
                if name in seen_names:
                    continue
                seen_names.add(name)

                aliases = " ".join(getattr(recipe, "aliases", []))
                d_name = getattr(recipe, "display_name", "") or ""
                desc = recipe.description or f"Declarative recipe for {name}"
                provs = (
                    list(recipe.providers.keys())
                    if getattr(recipe, "providers", None)
                    else ["recipe"]
                )
                prov_str = f"recipe ({', '.join(provs)})"

                content = f"{name} {d_name} {aliases} {desc}"
                corpus.append({
                    "name": recipe.name,
                    "description": desc,
                    "provider": prov_str,
                    "text": content.lower(),
                })
        except Exception as e:
            logger.debug(f"Failed to harvest recipes: {e}")

        # 2. Load Purpose Mappings
        try:
            from arjax.intelligence.suggest import PurposeSuggester
            suggester = PurposeSuggester()
            intent_terms = getattr(suggester, "intent_search_terms", {})
            for intent, apps in intent_terms.items():
                cat_text = intent.replace("-", " ")
                for app in apps:
                    app_name = app.lower().strip()
                    if app_name not in seen_names:
                        seen_names.add(app_name)
                        desc = f"Recommended app for {cat_text}"
                        content = f"{app} {intent} {cat_text}"
                        corpus.append({
                            "name": app,
                            "description": desc,
                            "provider": "repository",
                            "text": content.lower(),
                        })
        except Exception as e:
            logger.debug(f"Failed to harvest purpose mappings: {e}")

        # 3. Add Curated Alternatives and Mutual Peer References
        for alt_key, alts in self.POPULAR_ALTERNATIVES.items():
            all_peer_names = [a["name"] for a in alts]
            for alt in alts:
                alt_name = alt["name"].lower().strip()
                peers = " ".join([
                    p for p in all_peer_names if p.lower() != alt_name
                ])
                synonym = (
                    f"{alt['name']} {alt['desc']} {alt_key} "
                    f"alternative {peers}"
                )
                if alt_name not in seen_names:
                    seen_names.add(alt_name)
                    corpus.append({
                        "name": alt["name"],
                        "description": alt["desc"],
                        "provider": alt.get("prov", "repo"),
                        "text": synonym.lower(),
                    })
                else:
                    for p in corpus:
                        if p["name"].lower() == alt_name:
                            p["text"] += f" {alt_key} alternative {peers}"

        self.packages = corpus

    def _fit_engine(self) -> None:
        """Fit scikit-learn or pure-python TF-IDF vectorizer."""
        if not self.packages:
            return

        texts = [p["text"] for p in self.packages]

        if self.use_sklearn:
            try:
                self.vectorizer = TfidfVectorizer(
                    ngram_range=(1, 2),
                    stop_words="english",
                    sublinear_tf=True,
                )
                self.tfidf_matrix = self.vectorizer.fit_transform(texts)
                return
            except Exception as e:
                logger.debug(f"sklearn fit failed: {e}. Falling back.")
                self.use_sklearn = False

        self._fit_pure_python(texts)

    def _tokenize(self, text: str) -> List[str]:
        """Clean and tokenize text into words."""
        tokens = re.findall(r"\b[a-zA-Z0-9_\-+#]{2,}\b", text.lower())
        return [t for t in tokens if t not in self.DEFAULT_STOP_WORDS]

    def _fit_pure_python(self, texts: List[str]) -> None:
        """Build term frequency and inverse document frequency structures."""
        total_docs = len(texts)
        doc_freqs: Counter = Counter()
        tokenized_docs = []

        for text in texts:
            tokens = self._tokenize(text)
            tokenized_docs.append(tokens)
            unique_terms = set(tokens)
            doc_freqs.update(unique_terms)

        self._doc_frequencies = dict(doc_freqs)
        self._idf = {
            term: math.log((total_docs + 1) / (count + 1)) + 1.0
            for term, count in doc_freqs.items()
        }

        self._doc_vectors = []
        for tokens in tokenized_docs:
            if not tokens:
                self._doc_vectors.append({})
                continue
            tf = Counter(tokens)
            vec: Dict[str, float] = {}
            for term, count in tf.items():
                tf_val = 1.0 + math.log(count) if count > 0 else 0.0
                vec[term] = tf_val * self._idf.get(term, 1.0)
            norm = math.sqrt(sum(v * v for v in vec.values()))
            if norm > 0:
                vec = {k: v / norm for k, v in vec.items()}
            self._doc_vectors.append(vec)

    def recommend(
        self,
        query: str,
        top_n: Optional[int] = None,
        exclude_exact: bool = True,
    ) -> List[RecommendationResult]:
        """Compute and return top N related package recommendations."""
        limit = top_n or self.top_n
        clean_query = query.lower().strip()
        if not clean_query or not self.packages:
            return []

        curated_matches: List[RecommendationResult] = []
        norm_key = clean_query.replace(" ", "").replace("-", "")
        for key, alts in self.POPULAR_ALTERNATIVES.items():
            if key in clean_query or norm_key in key:
                for alt in alts:
                    if exclude_exact and alt["name"].lower() == clean_query:
                        continue
                    curated_matches.append(RecommendationResult(
                        name=alt["name"],
                        description=alt["desc"],
                        provider=alt.get("prov", "repo"),
                        similarity_score=0.96,
                    ))

        if curated_matches:
            seen = set()
            unique_curated = []
            for c in curated_matches:
                if c.name not in seen:
                    seen.add(c.name)
                    unique_curated.append(c)
            return unique_curated[:limit]

        has_sklearn = (
            self.use_sklearn
            and self.vectorizer is not None
            and self.tfidf_matrix is not None
        )
        if has_sklearn:
            return self._recommend_sklearn(clean_query, limit, exclude_exact)
        else:
            return self._recommend_pure_python(
                clean_query, limit, exclude_exact
            )

    def _recommend_sklearn(
        self, query: str, limit: int, exclude_exact: bool
    ) -> List[RecommendationResult]:
        """Calculate recommendations with scikit-learn cosine similarity."""
        query_vec = self.vectorizer.transform([query])
        scores = cosine_similarity(query_vec, self.tfidf_matrix).flatten()

        ranked_indices = scores.argsort()[::-1]
        results: List[RecommendationResult] = []
        seen = set()

        for idx in ranked_indices:
            score = float(scores[idx])
            if score < self.min_score:
                break
            pkg = self.packages[idx]
            name = pkg["name"]
            if name in seen:
                continue
            if exclude_exact and name.lower() == query:
                continue
            seen.add(name)
            results.append(RecommendationResult(
                name=name,
                description=pkg["description"],
                provider=pkg["provider"],
                similarity_score=round(score, 3),
            ))
            if len(results) >= limit:
                break

        return results

    def _recommend_pure_python(
        self, query: str, limit: int, exclude_exact: bool
    ) -> List[RecommendationResult]:
        """Calculate recommendations with pure-Python cosine similarity."""
        query_tokens = self._tokenize(query)
        if not query_tokens:
            return []

        tf = Counter(query_tokens)
        query_vec: Dict[str, float] = {}
        for term, count in tf.items():
            tf_val = 1.0 + math.log(count) if count > 0 else 0.0
            query_vec[term] = tf_val * self._idf.get(term, 1.0)
        norm = math.sqrt(sum(v * v for v in query_vec.values()))
        if norm > 0:
            query_vec = {k: v / norm for k, v in query_vec.items()}

        results: List[RecommendationResult] = []
        seen = set()

        for idx, doc_vec in enumerate(self._doc_vectors):
            pkg = self.packages[idx]
            name = pkg["name"]
            if name in seen:
                continue
            if exclude_exact and name.lower() == query:
                continue

            score = sum(
                val * doc_vec.get(term, 0.0)
                for term, val in query_vec.items()
            )
            if score >= self.min_score:
                seen.add(name)
                results.append(RecommendationResult(
                    name=name,
                    description=pkg["description"],
                    provider=pkg["provider"],
                    similarity_score=round(score, 3),
                ))

        results.sort(key=lambda r: r.similarity_score, reverse=True)
        return results[:limit]
