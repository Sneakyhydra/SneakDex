//! Language detection utility module.
//!
//! Uses `whatlang` crate to detect the language of a given text.

use whatlang::detect;

/// Detect the language of the given text and return its ISO 639-1 code (`en`, `fr`, etc.).
///
/// # Arguments
/// `text` - The input text to analyze.
///
/// # Returns
/// `Some(String)` with the language code if detected.
/// `None` if detection failed.
///
/// # Example
/// ```
/// let lang = detect_language("Hello, world!");
/// assert_eq!(lang.as_deref(), Some("en"));
/// ```
pub fn detect_language(text: &str) -> Option<String> {
    // `detect` returns an Option<Info>, which contains `lang()`
    detect(text).map(|info| info.lang().code().to_string())
}
