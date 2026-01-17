import click
import json
import sys
import tempfile
from pathlib import Path

@click.command()
@click.option('--action', type=click.Choice(['create', 'read', 'update', 'delete']), required=True, help='Action to perform')
def main(action):
    input_data = sys.stdin.read()
    data = json.loads(input_data)

    match action:
        case 'create':
            handle_create(data)
        case 'read':
            handle_read(data)
        case 'update':
            handle_update(data)
        case 'delete':
            handle_delete(data)

def handle_create(data):
    content = data.get("input", {}).get("content", "")
    with tempfile.NamedTemporaryFile(delete=False) as tmp:
        tmp_path = Path(tmp.name)
        tmp_path.write_text(content)
    
    print(json.dumps({"id": str(tmp_path), "content": content}))

def handle_read(data):
    path = Path(data.get("id"))
    if not path.exists():
        sys.exit(22)

    print(json.dumps({"id": str(path), "content": path.read_text()}))

def handle_update(data):
    path = Path(data.get("id"))
    content = data.get("input", {}).get("content", "")
    path.write_text(content)
    print(json.dumps({"id": str(path), "content": content}))

def handle_delete(data):
    Path(data.get("id")).unlink(missing_ok=True)

if __name__ == '__main__':
    main()
