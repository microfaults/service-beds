import os
import subprocess
import sys

# manage_replace.py is in service-beds/microservices-demo-go/
REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
ATROPOS_GO_PATH = os.path.abspath(os.path.join(REPO_ROOT, "atropos-go"))
SRC_DIR = os.path.join(os.path.dirname(__file__), "src")

def get_go_mods(start_dir):
    go_mods = []
    for root, dirs, files in os.walk(start_dir):
        if "go.mod" in files:
            go_mods.append(root)
            if "vendor" in dirs:
                dirs.remove("vendor")
    return go_mods

def run_command(cmd, cwd):
    print(f"Running: {' '.join(cmd)} in {cwd}")
    try:
        subprocess.run(cmd, cwd=cwd, check=True, shell=(os.name == 'nt'))
    except subprocess.CalledProcessError as e:
        print(f"Error running command: {e}")

def get_relative_path(mod_dir):
    rel_path = os.path.relpath(ATROPOS_GO_PATH, mod_dir)
    return rel_path.replace(os.sep, "/")

def run_vendor_script():
    script_dir = os.path.dirname(__file__)
    if os.name == 'nt':
        print("Running run_vendor.bat...")
        run_command(["run_vendor.bat"], script_dir)
    else:
        print("Running run_vendor.sh...")
        run_command(["chmod", "+x", "run_vendor.sh"], script_dir)
        run_command(["./run_vendor.sh"], script_dir)

def add_replace():
    mod_dirs = get_go_mods(SRC_DIR)
    for mod_dir in mod_dirs:
        rel_path = get_relative_path(mod_dir)
        print(f"Adding replace to {mod_dir}")
        run_command(["go", "mod", "edit", "-replace", f"git.ucsc.edu/microfaults/atropos-go={rel_path}"], mod_dir)
        run_command(["go", "mod", "tidy"], mod_dir)
    run_vendor_script()

def remove_replace():
    mod_dirs = get_go_mods(SRC_DIR)
    for mod_dir in mod_dirs:
        print(f"Removing replace from {mod_dir}")
        run_command(["go", "mod", "edit", "-dropreplace", "git.ucsc.edu/microfaults/atropos-go"], mod_dir)
        run_command(["go", "mod", "tidy"], mod_dir)
    run_vendor_script()

def sync_vendor():
    mod_dirs = get_go_mods(SRC_DIR)
    for mod_dir in mod_dirs:
        print(f"Tidying {mod_dir}")
        run_command(["go", "mod", "tidy"], mod_dir)
    run_vendor_script()

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python manage_replace.py [add|remove|sync]")
        sys.exit(1)
    
    cmd = sys.argv[1]
    if cmd == "add":
        add_replace()
    elif cmd == "remove":
        remove_replace()
    elif cmd == "sync":
        sync_vendor()
    else:
        print(f"Unknown command: {cmd}")
        sys.exit(1)
