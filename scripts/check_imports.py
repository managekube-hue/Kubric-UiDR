"""Validate all intra-kai imports reference real module paths."""
import pathlib, ast, sys

kai_root = pathlib.Path("kai")

errors = []
for f in sorted(kai_root.rglob("*.py")):
    src = f.read_text(encoding="utf-8")
    tree = ast.parse(src)
    for node in ast.walk(tree):
        if isinstance(node, ast.ImportFrom) and node.module:
            if node.module.startswith("kai."):
                parts = node.module.split(".")
                target = kai_root
                valid = True
                for part in parts[1:]:
                    candidate_file = target / (part + ".py")
                    candidate_pkg  = target / part / "__init__.py"
                    if candidate_file.exists():
                        target = candidate_file
                        break
                    elif candidate_pkg.exists():
                        target = target / part
                    else:
                        valid = False
                        break
                if not valid:
                    errors.append(f"{f}:{node.lineno}  bad import 'from {node.module}'")

if errors:
    print("IMPORT ERRORS:")
    for e in errors:
        print(f"  {e}")
    sys.exit(1)
else:
    print(f"ALL INTRA-KAI IMPORTS VALID")
