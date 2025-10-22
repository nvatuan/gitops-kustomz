package diff

import (
	"testing"
)

// TestCalcLineChangesFromDiffContent tests the line counting utility function
func TestCalcLineChangesFromDiffContent(t *testing.T) {
	tests := []struct {
		name            string
		diffContent     string
		expectedAdded   int
		expectedDeleted int
		expectedTotal   int
	}{
		{
			name:            "empty diff",
			diffContent:     "",
			expectedAdded:   0,
			expectedDeleted: 0,
			expectedTotal:   0,
		},
		{
			name: "simple additions and deletions",
			diffContent: `--- before	2025-10-23 00:52:21
+++ after	2025-10-23 00:52:21
@@ -1,3 +1,4 @@
 line1
- line2
- line3
+ line2_modified
+ line3
+ line4
`,
			expectedAdded:   3,
			expectedDeleted: 2,
			expectedTotal:   5,
		},
		{
			name: "only additions",
			diffContent: `--- before	2025-10-23 00:52:21
+++ after	2025-10-23 00:52:21
@@ -0,0 +1,2 @@
+ new line 1
+ new line 2
`,
			expectedAdded:   2,
			expectedDeleted: 0,
			expectedTotal:   2,
		},
		{
			name: "only deletions",
			diffContent: `--- before	2025-10-23 00:52:21
+++ after	2025-10-23 00:52:21
@@ -1,2 +0,0 @@
- old line 1
- old line 2
`,
			expectedAdded:   0,
			expectedDeleted: 2,
			expectedTotal:   2,
		},
		{
			name: "mixed changes with context",
			diffContent: `--- before	2025-10-23 00:52:21
+++ after	2025-10-23 00:52:21
@@ -1,4 +1,4 @@
 line1
- line2
+ line2_modified
 line3
- line4
+ line4_modified
 line5
`,
			expectedAdded:   2,
			expectedDeleted: 2,
			expectedTotal:   4,
		},
		{
			name: "no newline markers",
			diffContent: `--- before	2025-10-23 00:52:21
+++ after	2025-10-23 00:52:21
@@ -1,3 +1,4 @@
 line1
- line2
- line3
+ line2_modified
+ line3
+ line4
\\ No newline at end of file
`,
			expectedAdded:   3, // line2_modified, line3, line4
			expectedDeleted: 2, // line2, line3
			expectedTotal:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			added, deleted, total := CalcLineChangesFromDiffContent(tt.diffContent)

			if added != tt.expectedAdded {
				t.Errorf("CalcLineChangesFromDiffContent() added = %v, want %v", added, tt.expectedAdded)
			}
			if deleted != tt.expectedDeleted {
				t.Errorf("CalcLineChangesFromDiffContent() deleted = %v, want %v", deleted, tt.expectedDeleted)
			}
			if total != tt.expectedTotal {
				t.Errorf("CalcLineChangesFromDiffContent() total = %v, want %v", total, tt.expectedTotal)
			}

			// Verify that total equals added + deleted
			if total != added+deleted {
				t.Errorf("Total (%d) should equal added (%d) + deleted (%d)", total, added, deleted)
			}
		})
	}
}

// TestCalcLineChangesFromDiffContent_EdgeCases tests edge cases for the utility function
func TestCalcLineChangesFromDiffContent_EdgeCases(t *testing.T) {
	t.Run("lines starting with + or - but not diff markers", func(t *testing.T) {
		// Test that we only count lines that start with + or - (not + or - followed by space)
		diffContent := `--- before	2025-10-23 00:52:21
+++ after	2025-10-23 00:52:21
@@ -1,2 +1,2 @@
 line1
- line2
+ line2_modified
 line3
`
		added, deleted, total := CalcLineChangesFromDiffContent(diffContent)

		if added != 1 {
			t.Errorf("Expected 1 added line, got %d", added)
		}
		if deleted != 1 {
			t.Errorf("Expected 1 deleted line, got %d", deleted)
		}
		if total != 2 {
			t.Errorf("Expected 2 total lines, got %d", total)
		}
	})

	t.Run("lines with + or - in content but not at start", func(t *testing.T) {
		// Test that we don't count lines that have + or - in the middle
		diffContent := `--- before	2025-10-23 00:52:21
+++ after	2025-10-23 00:52:21
@@ -1,2 +1,2 @@
 line1
- line2
+ line2_modified
 line3
`
		added, deleted, total := CalcLineChangesFromDiffContent(diffContent)

		if added != 1 {
			t.Errorf("Expected 1 added line, got %d", added)
		}
		if deleted != 1 {
			t.Errorf("Expected 1 deleted line, got %d", deleted)
		}
		if total != 2 {
			t.Errorf("Expected 2 total lines, got %d", total)
		}
	})
}

// BenchmarkCalcLineChangesFromDiffContent benchmarks the utility function
func BenchmarkCalcLineChangesFromDiffContent(b *testing.B) {
	// Create a large diff content for benchmarking
	diffContent := `--- before	2025-10-23 00:52:21
+++ after	2025-10-23 00:52:21
@@ -1,100 +1,100 @@
 line1
-line2
+line2_modified
 line3
-line4
+line4_modified
 line5
-line6
+line6_modified
 line7
-line8
+line8_modified
 line9
-line10
+line10_modified
 line11
-line12
+line12_modified
 line13
-line14
+line14_modified
 line15
-line16
+line16_modified
 line17
-line18
+line18_modified
 line19
-line20
+line20_modified
 line21
-line22
+line22_modified
 line23
-line24
+line24_modified
 line25
-line26
+line26_modified
 line27
-line28
+line28_modified
 line29
-line30
+line30_modified
 line31
-line32
+line32_modified
 line33
-line34
+line34_modified
 line35
-line36
+line36_modified
 line37
-line38
+line38_modified
 line39
-line40
+line40_modified
 line41
-line42
+line42_modified
 line43
-line44
+line44_modified
 line45
-line46
+line46_modified
 line47
-line48
+line48_modified
 line49
-line50
+line50_modified
 line51
-line52
+line52_modified
 line53
-line54
+line54_modified
 line55
-line56
+line56_modified
 line57
-line58
+line58_modified
 line59
-line60
+line60_modified
 line61
-line62
+line62_modified
 line63
-line64
+line64_modified
 line65
-line66
+line66_modified
 line67
-line68
+line68_modified
 line69
-line70
+line70_modified
 line71
-line72
+line72_modified
 line73
-line74
+line74_modified
 line75
-line76
+line76_modified
 line77
-line78
+line78_modified
 line79
-line80
+line80_modified
 line81
-line82
+line82_modified
 line83
-line84
+line84_modified
 line85
-line86
+line86_modified
 line87
-line88
+line88_modified
 line89
-line90
+line90_modified
 line91
-line92
+line92_modified
 line93
-line94
+line94_modified
 line95
-line96
+line96_modified
 line97
-line98
+line98_modified
 line99
-line100
+line100_modified
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalcLineChangesFromDiffContent(diffContent)
	}
}
