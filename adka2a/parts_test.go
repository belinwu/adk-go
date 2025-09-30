package adka2a

import (
	"testing"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/genai"
)

func TestPartsTwoWayConversion(t *testing.T) {
	testCases := []struct {
		name                   string
		a2aPart                a2a.Part
		genaiPart              *genai.Part
		longRunningFunctionIDs []string
	}{
		{
			name:      "text",
			a2aPart:   a2a.TextPart{Text: "Hello"},
			genaiPart: &genai.Part{Text: "Hello"},
		},
		{
			name:      "thought",
			a2aPart:   a2a.TextPart{Text: "Hello", Metadata: map[string]any{toMetaKey("thought"): true}},
			genaiPart: &genai.Part{Text: "Hello", Thought: true},
		},
		{
			name: "file uri",
			a2aPart: a2a.FilePart{
				File: a2a.FileURI{URI: "ftp://cat.com", FileMeta: a2a.FileMeta{MimeType: "image/jpeg", Name: "cat.jpeg"}},
			},
			genaiPart: &genai.Part{
				FileData: &genai.FileData{FileURI: "ftp://cat.com", MIMEType: "image/jpeg", DisplayName: "cat.jpeg"},
			},
		},
		{
			name: "file bytes",
			a2aPart: a2a.FilePart{
				File: a2a.FileBytes{Bytes: "/w==", FileMeta: a2a.FileMeta{MimeType: "image/jpeg", Name: "cat.jpeg"}},
			},
			genaiPart: &genai.Part{
				InlineData: &genai.Blob{Data: []byte{0xfF}, MIMEType: "image/jpeg", DisplayName: "cat.jpeg"},
			},
		},
		{
			name: "function call",
			a2aPart: a2a.DataPart{
				Data: map[string]any{
					"id":   "get_weather",
					"args": map[string]any{"city": "Warsaw"},
					"name": "GetWeather",
				},
				Metadata: map[string]any{
					a2aDataPartMetaTypeKey:        a2aDataPartTypeFunctionCall,
					a2aDataPartMetaLongRunningKey: false,
				},
			},
			genaiPart: &genai.Part{
				FunctionCall: &genai.FunctionCall{
					ID:   "get_weather",
					Args: map[string]any{"city": "Warsaw"},
					Name: "GetWeather",
				},
			},
		},
		{
			name: "long running function call",
			a2aPart: a2a.DataPart{
				Data: map[string]any{
					"id":   "get_weather",
					"args": map[string]any{"city": "Warsaw"},
					"name": "GetWeather",
				},
				Metadata: map[string]any{
					a2aDataPartMetaTypeKey:        a2aDataPartTypeFunctionCall,
					a2aDataPartMetaLongRunningKey: true,
				},
			},
			genaiPart: &genai.Part{
				FunctionCall: &genai.FunctionCall{
					ID:   "get_weather",
					Args: map[string]any{"city": "Warsaw"},
					Name: "GetWeather",
				},
			},
			longRunningFunctionIDs: []string{"get_weather"},
		},
		{
			name: "function response",
			a2aPart: a2a.DataPart{
				Data: map[string]any{
					"id":         "get_weather",
					"scheduling": string(genai.FunctionResponseSchedulingInterrupt),
					"response":   map[string]any{"temperature": "7C"},
					"name":       "GetWeather",
				},
				Metadata: map[string]any{a2aDataPartMetaTypeKey: a2aDataPartTypeFunctionResponse},
			},
			genaiPart: &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					ID:         "get_weather",
					Scheduling: genai.FunctionResponseSchedulingInterrupt,
					Response:   map[string]any{"temperature": "7C"},
					Name:       "GetWeather",
				},
			},
		},
		{
			name: "code execution result",
			a2aPart: a2a.DataPart{
				Data:     map[string]any{"outcome": string(genai.OutcomeOK), "output": "4"},
				Metadata: map[string]any{a2aDataPartMetaTypeKey: a2aDataPartTypeCodeExecResult},
			},
			genaiPart: &genai.Part{
				CodeExecutionResult: &genai.CodeExecutionResult{
					Outcome: genai.OutcomeOK,
					Output:  "4",
				},
			},
		},
		{
			name: "code execution result",
			a2aPart: a2a.DataPart{
				Data:     map[string]any{"code": "print(2+2)", "language": string(genai.LanguagePython)},
				Metadata: map[string]any{a2aDataPartMetaTypeKey: a2aDataPartTypeCodeExecutableCode},
			},
			genaiPart: &genai.Part{
				ExecutableCode: &genai.ExecutableCode{
					Code:     "print(2+2)",
					Language: genai.LanguagePython,
				},
			},
		},
	}

	for _, tc := range testCases {
		toA2A, err := toA2AParts([]*genai.Part{tc.genaiPart}, tc.longRunningFunctionIDs)
		if err != nil {
			t.Fatalf("toA2AParts() failed: %v", err)
		}
		if diff := cmp.Diff(toA2A, []a2a.Part{tc.a2aPart}); diff != "" {
			t.Fatalf("toA2A failed (+want,-got)\nwant = %v\ngot = %v\ndiff = %s", tc.a2aPart, toA2A, diff)
		}

		toGenAI, err := toGenAIParts([]a2a.Part{tc.a2aPart})
		if err != nil {
			t.Fatalf("toGenAI() failed: %v", err)
		}
		if diff := cmp.Diff(toGenAI, []*genai.Part{tc.genaiPart}); diff != "" {
			t.Fatalf("toGenAI failed (+want,-got)\nwant = %v\ngot = %v\ndiff = %s", tc.a2aPart, toA2A, diff)
		}
	}
}

func TestPartsOneWayConversion(t *testing.T) {
	part := a2a.DataPart{Data: map[string]any{"arbitrary": "data"}}
	wantGenAI := &genai.Part{Text: `{"arbitrary":"data"}`}

	gotGenAI, err := toGenAIParts([]a2a.Part{part})
	if err != nil {
		t.Fatalf("toGenAI() failed: %v", err)
	}
	if diff := cmp.Diff(gotGenAI, []*genai.Part{wantGenAI}); diff != "" {
		t.Fatalf("toGenAI failed (+want,-got)\nwant = %v\ngot = %v\ndiff = %s", part, gotGenAI, diff)
	}

	wantA2A := a2a.TextPart{Text: `{"arbitrary":"data"}`}
	gotA2A, err := toA2AParts(gotGenAI, nil)
	if err != nil {
		t.Fatalf("toA2AParts() failed: %v", err)
	}
	if diff := cmp.Diff(gotA2A, []a2a.Part{wantA2A}); diff != "" {
		t.Fatalf("toA2A failed (+want,-got)\nwant = %v\ngot = %v\ndiff = %s", wantA2A, gotA2A, diff)
	}

}
