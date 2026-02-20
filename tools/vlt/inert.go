package main

import "regexp"

// maskPass is a function that masks one type of inert zone.
// Each pass receives the text (potentially already partially masked by
// earlier passes) and returns the text with its zone type masked.
type maskPass func(text string) string

// inertPasses is the ordered slice of mask functions.
// Each story adds its pass to this slice via an init() function or by
// calling registerMaskPass. Order matters:
// fenced code blocks first, then inline code, then comments, then math.
var inertPasses []maskPass

// registerMaskPass adds a masking pass. Called during init.
func registerMaskPass(p maskPass) {
	inertPasses = append(inertPasses, p)
}

// maskInertContent applies all registered masking passes in order.
// The result has the same byte length and line count as the input,
// but content inside inert zones is replaced with spaces (preserving newlines).
func maskInertContent(text string) string {
	for _, pass := range inertPasses {
		text = pass(text)
	}
	return text
}

// maskRegion replaces all non-newline characters in text[start:end] with spaces.
// Newlines are preserved so that line numbers remain stable.
func maskRegion(text []byte, start, end int) {
	for i := start; i < end; i++ {
		if text[i] != '\n' {
			text[i] = ' '
		}
	}
}

// fencedCodePattern matches the opening fence of a fenced code block:
// three or more backticks at the start of a line, optionally followed by a
// language identifier, then a newline.
var fencedCodePattern = regexp.MustCompile("(?m)^(```\\w*)\n")

// closingFencePattern matches a closing fence: three backticks at the start
// of a line (possibly followed by whitespace and then end-of-line or end-of-string).
var closingFencePattern = regexp.MustCompile("(?m)^```[ \t]*$")

// maskFencedCodeBlocks masks the content inside fenced code blocks (``` ... ```).
// The fence delimiters themselves are NOT masked.
// Unclosed fences at EOF: mask to end of file (matches Obsidian behavior).
func maskFencedCodeBlocks(text string) string {
	buf := []byte(text)
	pos := 0

	for pos < len(buf) {
		// Find the next opening fence
		loc := fencedCodePattern.FindIndex(buf[pos:])
		if loc == nil {
			break
		}

		// The content to mask starts after the opening fence line (after the \n)
		openEnd := pos + loc[1] // position right after the opening fence line's newline
		contentStart := openEnd

		// Find the closing fence starting from content area
		closeLoc := closingFencePattern.FindIndex(buf[contentStart:])
		if closeLoc == nil {
			// Unclosed fence: mask to end of file
			maskRegion(buf, contentStart, len(buf))
			break
		}

		// Mask from content start to the start of the closing fence line
		contentEnd := contentStart + closeLoc[0]
		maskRegion(buf, contentStart, contentEnd)

		// Move past the closing fence
		pos = contentStart + closeLoc[1]
	}

	return string(buf)
}

// doubleBacktickPattern matches inline code spans delimited by ``.
// Group 1 captures the content between the double-backtick delimiters.
// Must be applied before singleBacktickPattern to avoid partial matches.
var doubleBacktickPattern = regexp.MustCompile("``([^`\\n]+)``")

// singleBacktickPattern matches inline code spans delimited by `.
// Group 1 captures the content between the single-backtick delimiters.
var singleBacktickPattern = regexp.MustCompile("`([^`\\n]+)`")

// maskInlineCode masks the content inside inline code spans (` ... ` and `` ... ``).
// The backtick delimiters themselves are NOT masked, only the content between them.
// This pass runs AFTER fenced code blocks so that backticks already masked
// inside fenced blocks (replaced with spaces) do not trigger false matches.
// Double-backtick spans are processed first so that `` ` `` is handled correctly.
func maskInlineCode(text string) string {
	buf := []byte(text)

	// Pass 1: mask double-backtick spans first.
	for _, loc := range doubleBacktickPattern.FindAllSubmatchIndex(buf, -1) {
		// loc[2], loc[3] = start, end of group 1 (content)
		maskRegion(buf, loc[2], loc[3])
	}

	// Pass 2: mask single-backtick spans.
	// After pass 1, content in double-backtick spans is spaces, so the
	// single-backtick pass will not find backtick pairs inside them.
	for _, loc := range singleBacktickPattern.FindAllSubmatchIndex(buf, -1) {
		maskRegion(buf, loc[2], loc[3])
	}

	return string(buf)
}

// obsidianCommentPattern matches Obsidian-style comments: %% content %%.
// Uses (?s) (DOTALL) so that . matches newlines, enabling multiline comments.
// The match is non-greedy (*?) to handle multiple comments in the same text.
var obsidianCommentPattern = regexp.MustCompile(`(?s)%%(.+?)%%`)

// maskObsidianComments masks the content inside Obsidian comments (%% ... %%).
// The %% delimiters themselves are preserved; only the content between them is
// replaced with spaces (newlines preserved). This pass runs AFTER fenced code
// blocks and inline code, so %% inside already-masked code zones will not
// trigger false comment boundaries.
func maskObsidianComments(text string) string {
	buf := []byte(text)

	for _, loc := range obsidianCommentPattern.FindAllSubmatchIndex(buf, -1) {
		// loc[2], loc[3] = start, end of group 1 (content between %% delimiters)
		maskRegion(buf, loc[2], loc[3])
	}

	return string(buf)
}

// htmlCommentPattern matches HTML comments: <!-- content -->.
// Uses (?s) (DOTALL) so that . matches newlines, enabling multiline comments.
// The match is non-greedy (*?) to handle multiple comments in the same text.
var htmlCommentPattern = regexp.MustCompile(`(?s)<!--(.*?)-->`)

// maskHTMLComments masks the content inside HTML comments (<!-- ... -->).
// The <!-- and --> delimiters themselves are preserved; only the content
// between them is replaced with spaces (newlines preserved). This pass runs
// AFTER fenced code blocks, inline code, and Obsidian comments, so <!-- inside
// already-masked zones will not trigger false comment boundaries.
func maskHTMLComments(text string) string {
	buf := []byte(text)

	for _, loc := range htmlCommentPattern.FindAllSubmatchIndex(buf, -1) {
		// loc[2], loc[3] = start, end of group 1 (content between <!-- and -->)
		maskRegion(buf, loc[2], loc[3])
	}

	return string(buf)
}

// displayMathPattern matches display math blocks: $$ content $$.
// Uses (?s) (DOTALL) so that . matches newlines, enabling multiline blocks.
// The match is non-greedy (*?) to handle multiple display math blocks.
// Group 1 captures the content between the $$ delimiters.
var displayMathPattern = regexp.MustCompile(`(?s)\$\$(.+?)\$\$`)

// maskDisplayMath masks the content inside display math blocks ($$ ... $$).
// The $$ delimiters themselves are preserved; only the content between them is
// replaced with spaces (newlines preserved). This pass runs AFTER code blocks,
// inline code, and comments, so $$ inside already-masked zones will not
// trigger false math boundaries.
func maskDisplayMath(text string) string {
	buf := []byte(text)

	for _, loc := range displayMathPattern.FindAllSubmatchIndex(buf, -1) {
		// loc[2], loc[3] = start, end of group 1 (content between $$ delimiters)
		maskRegion(buf, loc[2], loc[3])
	}

	return string(buf)
}

// inlineMathPattern matches inline math: $content$.
// Requires the character after the opening $ to be a non-space, non-$ character,
// and the character before the closing $ to be a non-space, non-$ character.
// This prevents matching dollar amounts like $50 (no closing $) or
// spaced constructs like $ text $ (Obsidian also requires non-space).
// Does not cross newlines (inline math is single-line).
// Group 1 captures the content between the $ delimiters.
var inlineMathPattern = regexp.MustCompile(`\$([^\s$][^$\n]*?[^\s$])\$`)

// maskInlineMath masks the content inside inline math spans ($ ... $).
// The $ delimiters themselves are preserved; only the content between them is
// replaced with spaces. This pass runs AFTER display math so that $$ is not
// partially consumed by the inline math pattern.
func maskInlineMath(text string) string {
	buf := []byte(text)

	for _, loc := range inlineMathPattern.FindAllSubmatchIndex(buf, -1) {
		// loc[2], loc[3] = start, end of group 1 (content between $ delimiters)
		maskRegion(buf, loc[2], loc[3])
	}

	return string(buf)
}

func init() {
	// Order matters: fenced code blocks first, then inline code, then comments,
	// then math. Display math before inline math so $$ is not consumed as $.
	registerMaskPass(maskFencedCodeBlocks)
	registerMaskPass(maskInlineCode)
	registerMaskPass(maskObsidianComments)
	registerMaskPass(maskHTMLComments)
	registerMaskPass(maskDisplayMath)
	registerMaskPass(maskInlineMath)
}
