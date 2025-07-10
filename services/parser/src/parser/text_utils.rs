//! Utilities for text processing and cleaning.
//!
//! This module provides helper functions to normalize and clean text efficiently.

use once_cell::sync::Lazy;
use regex::Regex;

/// Precompiled regex to match one or more whitespace characters.
static RE_WHITESPACE: Lazy<Regex> = Lazy::new(|| {
    Regex::new(r"\s+").expect("Failed to compile whitespace regex")
});

/// Cleans and normalizes a string by collapsing all whitespace.
///
/// Trims leading and trailing whitespace, and replaces all internal
/// sequences of whitespace (spaces, tabs, newlines) with a single space.
///
/// # Arguments
///
/// `text` — The text to clean.
///
/// # Returns
///
/// A cleaned, single-spaced string.
///
/// # Example
///
/// ```
/// let cleaned = clean_text("   Hello   world \n\n how  are you?   ");
/// assert_eq!(cleaned, "Hello world how are you?");
/// ```
pub fn clean_text(text: &str) -> String {
    RE_WHITESPACE.replace_all(text.trim(), " ").to_string()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_clean_text_basic() {
        assert_eq!(clean_text("hello world"), "hello world");
    }

    #[test]
    fn test_clean_text_whitespace() {
        assert_eq!(clean_text("  hello   world  "), "hello world");
    }

    #[test]
    fn test_clean_text_newlines() {
        assert_eq!(clean_text("hello\n\nworld"), "hello world");
    }

    #[test]
    fn test_clean_text_tabs() {
        assert_eq!(clean_text("hello\tworld"), "hello world");
    }

    #[test]
    fn test_clean_text_mixed() {
        assert_eq!(clean_text("  hello \n\n  world \t test  "), "hello world test");
    }

    #[test]
    fn test_clean_text_empty() {
        assert_eq!(clean_text(""), "");
    }

    #[test]
    fn test_clean_text_only_whitespace() {
        assert_eq!(clean_text("   \n\t  "), "");
    }
}
