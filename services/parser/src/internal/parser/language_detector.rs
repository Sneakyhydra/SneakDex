//! Language detection utility module.
//!
//! Uses the `whatlang` crate to detect the language of a given text
//! and maps it to PostgreSQL full-text search configurations.

use whatlang::detect;

/// Detect the language of the given text and return its ISO 639-1 code (`en`, `fr`, etc.).
///
/// # Arguments
/// `text` - The input text to analyze.
///
/// # Returns
/// `Some(String)` with the language code if detected and confidence > 0.5.
/// `None` if detection failed or text is too short.
///
/// # Example
/// ```
/// let lang = detect_language("Hello, world!");
/// assert_eq!(lang.as_deref(), Some("en"));
/// ```
pub fn detect_language(text: &str) -> Option<String> {
    let text = text.trim();

    if text.len() < 20 {
        return None;
    }

    let info = detect(text)?;
    if info.confidence() > 0.5 {
        Some(info.lang().code().to_string())
    } else {
        None
    }
}

/// Maps ISO 639-1 or ISO 639-2 language codes to PostgreSQL FTS configurations.
///
/// Falls back to `"simple"` if no specific configuration exists.
///
/// # Arguments
/// `lang` - ISO 639-1 or 639-2 code.
///
/// # Returns
/// A PostgreSQL-compatible text search configuration.
pub fn map_lang_to_pg(lang: &str) -> &str {
    match lang {
        "en" | "eng" => "english",
        // "de" | "deu" => "german",
        // "fr" | "fra" => "french",
        // "ru" | "rus" => "russian",
        // "es" | "spa" => "spanish",
        // "it" | "ita" => "italian",
        // "pt" | "por" => "portuguese",
        // "nl" | "nld" => "dutch",
        // "sv" | "swe" => "swedish",
        // "fi" | "fin" => "finnish",
        // "no" | "nor" => "norwegian",
        // "da" | "dan" => "danish",
        // "hu" | "hun" => "hungarian",
        // "ro" | "ron" | "rum" => "romanian",
        // "tr" | "tur" => "turkish",
        // "bg" | "bul" => "bulgarian",
        // "ar" | "ara" => "arabic",
        // "cs" | "ces" | "cze" => "czech",
        // "el" | "gre" | "ell" => "greek",
        // "zh" | "zho" | "chi" => "chinese", // Postgres does not support Chinese natively; may need extensions
        // "ja" | "jpn" => "japanese",        // same
        _ => "simple", // fallback
    }
}
