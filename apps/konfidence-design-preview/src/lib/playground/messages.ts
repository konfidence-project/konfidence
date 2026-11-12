interface CompileRequest {
  id: number;
  source: string;
}
interface SourceDiagnostic {
  from: number;
  to: number;
  severity: "warning" | "error";
  message: string;
}
type ExampleCompilation =
  | { code: string; diagnostics: readonly SourceDiagnostic[]; error?: never }
  | { code?: never; diagnostics: readonly SourceDiagnostic[]; error: string };
type CompileResult = ExampleCompilation & { id: number };
type FrameResult =
  | { type: "focus-search" }
  | { type: "ready" }
  | { type: "rendered" }
  | { type: "error"; message: string };

export type { CompileRequest, CompileResult, ExampleCompilation, FrameResult, SourceDiagnostic };
