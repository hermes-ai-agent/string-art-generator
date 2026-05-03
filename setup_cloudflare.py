#!/usr/bin/env python3
"""
Cloudflare Pages Setup - Interactive
Automates GitHub secrets and deployment after manual token creation.
"""

import subprocess
import sys
import os

def run_cmd(cmd, check=True):
    """Run shell command and return output"""
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True, check=check)
    return result.stdout.strip()

def main():
    print("=== Cloudflare Pages Setup ===\n")
    
    # Step 1: Manual token creation
    print("Step 1: Create API Token")
    print("------------------------")
    print("1. Open: https://dash.cloudflare.com/profile/api-tokens")
    print("2. Click 'Create Token' → 'Get started' (Custom token)")
    print("3. Token name: GitHub Actions - Pages Deploy")
    print("4. Permissions:")
    print("   - Resource: Account")
    print("   - Permission: Cloudflare Pages")
    print("   - Level: Edit")
    print("5. Click 'Continue to summary' → 'Create Token'")
    print("6. COPY the token (shown only once!)\n")
    
    cf_token = input("Paste your Cloudflare API token here: ").strip()
    
    if not cf_token:
        print("Error: Token cannot be empty")
        sys.exit(1)
    
    print("\n✓ Token received\n")
    
    # Step 2: Set GitHub secrets
    print("Step 2: Setting up GitHub secrets...")
    print("------------------------------------")
    
    try:
        run_cmd(f"gh secret set CLOUDFLARE_API_TOKEN -b '{cf_token}' -R hermes-ai-agent/string-art-generator")
        run_cmd("gh secret set CLOUDFLARE_ACCOUNT_ID -b 'ed771e694c6365ea42180a5c54aadf6a' -R hermes-ai-agent/string-art-generator")
        print("✓ GitHub secrets configured\n")
    except subprocess.CalledProcessError as e:
        print(f"Error setting GitHub secrets: {e}")
        sys.exit(1)
    
    # Step 3: Commit and push workflow
    print("Step 3: Committing workflow...")
    print("-------------------------------")
    
    os.chdir(os.path.expanduser("~/string-art"))
    
    try:
        run_cmd("git add .github/workflows/deploy.yml")
        run_cmd('git commit -m "Add Cloudflare Pages deployment workflow"')
        run_cmd("git push")
        print("✓ Workflow pushed to GitHub\n")
    except subprocess.CalledProcessError as e:
        if "nothing to commit" in e.stderr:
            print("✓ Workflow already committed\n")
        else:
            print(f"Error committing: {e}")
            sys.exit(1)
    
    # Step 4: Instructions for Cloudflare Pages project
    print("Step 4: Create Cloudflare Pages project")
    print("----------------------------------------")
    print("Go to: https://dash.cloudflare.com/ed771e694c6365ea42180a5c54aadf6a/pages/new")
    print("1. Connect to Git → Select 'hermes-ai-agent/string-art-generator'")
    print("2. Build settings:")
    print("   - Framework preset: None")
    print("   - Build command: (leave empty)")
    print("   - Build output directory: docs")
    print("3. Click 'Save and Deploy'\n")
    
    print("After first deploy, GitHub Actions will handle future deployments automatically.")
    print("\n=== Setup Complete ===")

if __name__ == "__main__":
    main()
