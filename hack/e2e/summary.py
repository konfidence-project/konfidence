import sys
from pathlib import Path
import xml.etree.ElementTree as ET

report = Path(sys.argv[1])
print("## Example-app e2e")
if not report.is_file():
    print("No JUnit report was produced. Check the bootstrap and test steps.")
    sys.exit(0)

try:
    cases = ET.parse(report).getroot().findall(".//testcase")
except ET.ParseError:
    print("The JUnit report could not be parsed. See the e2e-results artifact.")
    sys.exit(0)

print("\n| Test | Result | Seconds |\n|---|---|---:|")
for case in cases:
    if case.find("failure") is not None or case.find("error") is not None:
        result = "Failed"
    elif case.find("skipped") is not None:
        result = "Skipped"
    else:
        result = "Passed"
    name = case.get("name", "").replace("|", "\\|").replace("\n", " ")
    print(f"| {name} | {result} | {case.get('time', '')} |")
