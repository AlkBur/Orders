package ui

import (
	"html/template"
	"testing"
)

func TestMarkMatches(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		words []string
		want  template.HTML
	}{
		{
			name:  "empty text",
			text:  "",
			words: []string{"альфа"},
			want:  "",
		},
		{
			name:  "empty words",
			text:  "ООО Альфа",
			words: nil,
			want:  "ООО Альфа",
		},
		{
			name:  "no match",
			text:  "ООО Ромашка",
			words: []string{"альфа"},
			want:  "ООО Ромашка",
		},
		{
			name:  "single word",
			text:  "ООО Альфа",
			words: []string{"альфа"},
			want:  `ООО <mark>Альфа</mark>`,
		},
		{
			name:  "case insensitive",
			text:  "oooo alfa",
			words: []string{"ALFA"},
			want:  `oooo <mark>alfa</mark>`,
		},
		{
			name:  "multiple words separate positions",
			text:  "Альфа Иванов",
			words: []string{"альфа", "иванов"},
			want:  `<mark>Альфа</mark> <mark>Иванов</mark>`,
		},
		{
			name:  "word order in list irrelevant",
			text:  "Альфа Иванов",
			words: []string{"иванов", "альфа"},
			want:  `<mark>Альфа</mark> <mark>Иванов</mark>`,
		},
		{
			name:  "overlapping words merge into one mark",
			text:  "Альфа",
			words: []string{"альфа", "а"},
			want:  `<mark>Альфа</mark>`,
		},
		{
			name:  "overlapping ranges stay flat without nesting",
			text:  "Альфа бета альфа",
			words: []string{"альфа", "фа бета"},
			want:  `<mark>Альфа бета</mark> <mark>альфа</mark>`,
		},
		{
			name:  "several occurrences of one word",
			text:  "Альфа и альфа",
			words: []string{"альфа"},
			want:  `<mark>Альфа</mark> и <mark>альфа</mark>`,
		},
		{
			name:  "substring at text start",
			text:  "Альфа старт",
			words: []string{"альфа"},
			want:  `<mark>Альфа</mark> старт`,
		},
		{
			name:  "substring at text end",
			text:  "конец Альфа",
			words: []string{"альфа"},
			want:  `конец <mark>Альфа</mark>`,
		},
		{
			name:  "whitespace words ignored",
			text:  "Альфа",
			words: []string{"   ", ""},
			want:  "Альфа",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MarkMatches(tt.text, tt.words)
			if got != tt.want {
				t.Fatalf("MarkMatches(%q, %v) = %q, want %q", tt.text, tt.words, got, tt.want)
			}
		})
	}
}

// TestMarkMatchesHTMLEscaping — безопасность template.HTML: исходный текст
// экранируется целиком, <mark> добавляется только хелпером. Готовый HTML
// из text не должен проходить насквозь.
func TestMarkMatchesHTMLEscaping(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		words []string
		want  template.HTML
	}{
		{
			name:  "script tag escaped without match",
			text:  `<script>alert(1)</script>`,
			words: []string{"альфа"},
			want:  `&lt;script&gt;alert(1)&lt;/script&gt;`,
		},
		{
			name:  "script tag escaped with match around it",
			text:  `ООО "Альфа" <script>alert(1)</script>`,
			words: []string{"альфа"},
			want:  `ООО &#34;<mark>Альфа</mark>&#34; &lt;script&gt;alert(1)&lt;/script&gt;`,
		},
		{
			name:  "match inside html-looking content",
			text:  `<b onclick="x()">Альфа</b>`,
			words: []string{"альфа"},
			want:  `&lt;b onclick=&#34;x()&#34;&gt;<mark>Альфа</mark>&lt;/b&gt;`,
		},
		{
			name:  "ampersand and quotes escaped",
			text:  `Иванов & Co "фирма"`,
			words: []string{"иванов"},
			want:  `<mark>Иванов</mark> &amp; Co &#34;фирма&#34;`,
		},
		{
			name:  "italic tag not created from text",
			text:  `Иванов</mark><script>x()</script>`,
			words: []string{"иванов"},
			want:  `<mark>Иванов</mark>&lt;/mark&gt;&lt;script&gt;x()&lt;/script&gt;`,
		},
		{
			name:  "single quote escaped",
			text:  `ООО 'Альфа'`,
			words: []string{"альфа"},
			want:  `ООО &#39;<mark>Альфа</mark>&#39;`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MarkMatches(tt.text, tt.words)
			if got != tt.want {
				t.Fatalf("MarkMatches(%q, %v) = %q, want %q", tt.text, tt.words, got, tt.want)
			}
		})
	}
}

// MarkMatches должен оставаться детерминированным и не паниковать на
// пустом тексте и на любых комбинациях слов.
func TestMarkMatchesNoPanic(t *testing.T) {
	inputs := []string{"", "просто текст", "привет", "x", "ёлка"}
	wordLists := [][]string{nil, {}, {""}, {" "}, {"привет", ""}, {"ёлка"}, {"Ёлка"}}
	for _, text := range inputs {
		for _, words := range wordLists {
			MarkMatches(text, words)
		}
	}
}
