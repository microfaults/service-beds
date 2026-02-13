import os
import re

manifest_dir = 'kubernetes-manifests'

def migrate_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    # Replace gRPC probes with HTTP probes
    # Regex explains:
    # (\n\s+) captures the newline and indentation before 'grpc:'
    # grpc: matches the literal string
    # \n\s+port: (\d+) matches the port line and captures the port number
    
    # We want to replace it with:
    # \1httpGet:
    # \1  path: /_healthz
    # \1  port: \2
    
    # Python regex replacement
    pattern = r'(\n\s+)grpc:\s*\n\s+port:\s+(\d+)'
    
    def replacer(match):
        indent = match.group(1) # includes \n
        base_indent = indent.replace('\n', '')
        port = match.group(2)
        return (f"{indent}httpGet:"
                f"\n{base_indent}  path: /_healthz"
                f"\n{base_indent}  port: {port}")

    new_content = re.sub(pattern, replacer, content)

    # Replace Service port name 'grpc' with 'http'
    new_content = new_content.replace('name: grpc', 'name: http')

    if new_content != content:
        print(f"Migrated {filepath}")
        with open(filepath, 'w') as f:
            f.write(new_content)

def main():
    if not os.path.exists(manifest_dir):
        print(f"Directory {manifest_dir} not found")
        return

    for filename in os.listdir(manifest_dir):
        if filename.endswith('.yaml'):
            migrate_file(os.path.join(manifest_dir, filename))

if __name__ == "__main__":
    main()
