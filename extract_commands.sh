#!/bin/bash

# Script that extracts and runs the setup commands from README.md
# This script parses the README.md file and extracts all bash commands within code blocks

README_PATH="./README.md"
OUTPUT_FILE="./acp_commands.sh"

echo "#!/bin/bash" > $OUTPUT_FILE
echo "" >> $OUTPUT_FILE
echo "# Commands extracted from $README_PATH" >> $OUTPUT_FILE
echo "# Generated on $(date)" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE
echo "# Set -e to exit on error" >> $OUTPUT_FILE
echo "set -e" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE
echo "# Add a function to check if we should continue after each step" >> $OUTPUT_FILE
echo "continue_prompt() {" >> $OUTPUT_FILE
echo "  read -p \"Press Enter to continue to the next command, or Ctrl+C to exit...\" dummy" >> $OUTPUT_FILE
echo "  echo \"\"" >> $OUTPUT_FILE
echo "}" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE

# Extract all bash command blocks that have actual commands
in_code_block=false
code_block_type=""
current_block=""
multiline_command=false
multiline_content=""

while IFS= read -r line; do
    # Check for code block start
    if [[ "$line" =~ ^'```'(.*)$ ]]; then
        block_type="${BASH_REMATCH[1]}"
        if [[ "$block_type" == "bash" ]]; then
            in_code_block=true
            code_block_type="bash"
            current_block=""
            multiline_command=false
            multiline_content=""
        fi
        continue
    fi
    
    # Check for code block end
    if [[ "$line" == '```' && "$in_code_block" == true ]]; then
        in_code_block=false
        
        # Process the entire block if it's a valid command block
        if [[ -n "$current_block" ]]; then
            # Filter out blocks that aren't actual commands
            if [[ ! "$current_block" =~ ^[[:space:]]*[A-Za-z0-9_-]+[[:space:]]+[A-Za-z0-9_-]+[[:space:]]+[A-Za-z0-9_-]+ ]] && 
               [[ ! "$current_block" =~ ^(NAME|NAMESPACE|STATUS|TYPE|REASON|AGE|FROM|MESSAGE|----|Output:) ]] &&
               [[ "$current_block" =~ (kind|kubectl|echo|export) ]]; then
                
                # Process multiline echo commands differently
                in_multiline_echo=false
                yaml_content=""
                resource_kind=""
                resource_name=""
                
                # Split block into lines for processing
                while IFS= read -r cmd; do
                    # Skip lines that look like outputs
                    if [[ "$cmd" =~ ^(NAME|NAMESPACE|STATUS|TYPE|REASON|AGE|FROM|MESSAGE|----) ]] || 
                       [[ "$cmd" =~ ^[[:space:]]*[0-9]+[[:space:]] ]] || 
                       [[ "$cmd" =~ ^\{.*\}$ ]] || 
                       [[ "$cmd" =~ ^[[:space:]]*\} ]] ||
                       [[ "$cmd" =~ ^[[:space:]]*\> ]]; then
                        continue
                    fi
                    
                    # Skip blank lines
                    if [[ -z "$cmd" ]]; then
                        continue
                    fi

                    # Skip lines that start with $ (shell prompt)
                    if [[ "$cmd" =~ ^\$ ]]; then
                        cmd="${cmd#$ }"
                    fi
                    
                    # Skip diagram notation
                    if [[ "$cmd" =~ ^graph|^flowchart|^subgraph ]]; then
                        continue
                    fi
                    
                    # Check for start of a multiline echo command (YAML creation)
                    if [[ "$cmd" =~ ^echo[[:space:]]*\'apiVersion: ]]; then
                        in_multiline_echo=true
                        yaml_content="$cmd"
                        continue
                    fi
                    
                    # Process lines that are part of a multiline echo
                    if [[ "$in_multiline_echo" == true ]]; then
                        yaml_content="$yaml_content"$'\n'"$cmd"
                        
                        # Extract resource kind and name for better output
                        if [[ "$cmd" =~ ^[[:space:]]*kind:[[:space:]]*([A-Za-z]+) ]]; then
                            resource_kind="${BASH_REMATCH[1]}"
                        fi
                        if [[ "$cmd" =~ ^[[:space:]]*[[:space:]]*name:[[:space:]]*([A-Za-z0-9_-]+) ]]; then
                            resource_name="${BASH_REMATCH[1]}"
                        fi
                        
                        # Check if we've reached the end of the multiline echo
                        if [[ "$cmd" =~ \'.*\|.*kubectl.*apply ]]; then
                            in_multiline_echo=false
                            
                            # Process the full echo command now that we have all of it
                            if [[ -n "$resource_kind" && -n "$resource_name" ]]; then
                                echo "echo \"Running: Creating $resource_kind $resource_name resource...\"" >> $OUTPUT_FILE
                                
                                # Add appropriate wait logic based on resource type
                                if [[ "$resource_kind" == "LLM" ]]; then
                                    echo "$yaml_content" >> $OUTPUT_FILE
                                    echo "echo \"Waiting for LLM $resource_name to initialize...\"" >> $OUTPUT_FILE
                                    echo "for i in {1..10}; do" >> $OUTPUT_FILE
                                    echo "  if kubectl get llm $resource_name -o jsonpath='{.status.ready}' 2>/dev/null | grep -q 'true'; then" >> $OUTPUT_FILE
                                    echo "    echo \"LLM $resource_name is ready!\"" >> $OUTPUT_FILE
                                    echo "    break" >> $OUTPUT_FILE
                                    echo "  fi" >> $OUTPUT_FILE
                                    echo "  sleep 2" >> $OUTPUT_FILE
                                    echo "  echo -n \".\"" >> $OUTPUT_FILE
                                    echo "done" >> $OUTPUT_FILE
                                    echo "echo \"\"" >> $OUTPUT_FILE
                                elif [[ "$resource_kind" == "Agent" ]]; then
                                    echo "$yaml_content" >> $OUTPUT_FILE
                                    echo "echo \"Waiting for Agent $resource_name to initialize...\"" >> $OUTPUT_FILE
                                    echo "for i in {1..10}; do" >> $OUTPUT_FILE
                                    echo "  if kubectl get agent $resource_name -o jsonpath='{.status.ready}' 2>/dev/null | grep -q 'true'; then" >> $OUTPUT_FILE
                                    echo "    echo \"Agent $resource_name is ready!\"" >> $OUTPUT_FILE
                                    echo "    break" >> $OUTPUT_FILE
                                    echo "  fi" >> $OUTPUT_FILE
                                    echo "  sleep 2" >> $OUTPUT_FILE
                                    echo "  echo -n \".\"" >> $OUTPUT_FILE
                                    echo "done" >> $OUTPUT_FILE
                                    echo "echo \"\"" >> $OUTPUT_FILE
                                elif [[ "$resource_kind" == "MCPServer" ]]; then
                                    echo "$yaml_content" >> $OUTPUT_FILE
                                    echo "echo \"Waiting for MCPServer $resource_name to initialize...\"" >> $OUTPUT_FILE
                                    echo "for i in {1..10}; do" >> $OUTPUT_FILE
                                    echo "  if kubectl get mcpserver $resource_name -o jsonpath='{.status.ready}' 2>/dev/null | grep -q 'true'; then" >> $OUTPUT_FILE
                                    echo "    echo \"MCPServer $resource_name is ready!\"" >> $OUTPUT_FILE
                                    echo "    break" >> $OUTPUT_FILE
                                    echo "  fi" >> $OUTPUT_FILE
                                    echo "  sleep 2" >> $OUTPUT_FILE
                                    echo "  echo -n \".\"" >> $OUTPUT_FILE
                                    echo "done" >> $OUTPUT_FILE
                                    echo "echo \"\"" >> $OUTPUT_FILE
                                elif [[ "$resource_kind" == "Task" ]]; then
                                    echo "$yaml_content" >> $OUTPUT_FILE
                                    echo "echo \"Waiting for Task $resource_name to complete...\"" >> $OUTPUT_FILE
                                    echo "for i in {1..15}; do" >> $OUTPUT_FILE
                                    echo "  status=\$(kubectl get task $resource_name -o jsonpath='{.status.phase}' 2>/dev/null || echo \"Pending\")" >> $OUTPUT_FILE
                                    echo "  if [[ \"\$status\" == \"FinalAnswer\" ]]; then" >> $OUTPUT_FILE
                                    echo "    echo \"Task $resource_name completed successfully!\"" >> $OUTPUT_FILE
                                    echo "    echo \"Result:\"" >> $OUTPUT_FILE
                                    echo "    kubectl get task $resource_name -o jsonpath='{.status.output}'" >> $OUTPUT_FILE
                                    echo "    echo \"\"" >> $OUTPUT_FILE
                                    echo "    break" >> $OUTPUT_FILE
                                    echo "  fi" >> $OUTPUT_FILE
                                    echo "  sleep 2" >> $OUTPUT_FILE
                                    echo "  echo -n \".\"" >> $OUTPUT_FILE
                                    echo "done" >> $OUTPUT_FILE
                                    echo "echo \"\"" >> $OUTPUT_FILE
                                else
                                    echo "$yaml_content" >> $OUTPUT_FILE
                                fi
                            else
                                # If we couldn't determine the resource type/name, just apply it
                                echo "echo \"Running: Applying YAML resource\"" >> $OUTPUT_FILE
                                echo "$yaml_content" >> $OUTPUT_FILE
                            fi
                            echo "continue_prompt" >> $OUTPUT_FILE
                            echo "" >> $OUTPUT_FILE
                        fi
                        continue
                    fi
                    
                    # For normal commands, just add them to the output
                    echo "echo \"Running: $cmd\"" >> $OUTPUT_FILE
                    echo "$cmd" >> $OUTPUT_FILE
                    echo "continue_prompt" >> $OUTPUT_FILE
                    echo "" >> $OUTPUT_FILE
                done <<< "$current_block"
            fi
        fi
        
        current_block=""
        code_block_type=""
        continue
    fi
    
    # Collect code block content
    if [[ "$in_code_block" == true && "$code_block_type" == "bash" ]]; then
        current_block+="$line"$'\n'
    fi
done < "$README_PATH"

# Process code blocks inside "echo" multi-line strings
# These are YAML blocks that are piped to kubectl
extract_echo_blocks() {
    local line="$1"
    if [[ "$line" =~ ^echo[[:space:]]*\'(apiVersion:.*)\'[[:space:]]*\|[[:space:]]*kubectl[[:space:]]apply[[:space:]]-f[[:space:]]-$ ]]; then
        # Found an echo with YAML content piped to kubectl apply
        local yaml_content="${BASH_REMATCH[1]}"
        
        # Skip if this is from a <details> or other non-primary example
        # Check for empty or incomplete YAML content
        if [[ "$yaml_content" =~ "spec:" && ! "$yaml_content" =~ "name:" ]]; then
            return 0
        fi
        
        # Try to extract resource kind and name from the YAML content
        local resource_kind=""
        local resource_name=""
        
        while IFS= read -r yaml_line; do
            if [[ "$yaml_line" =~ ^kind:[[:space:]]*([A-Za-z]+) ]]; then
                resource_kind="${BASH_REMATCH[1]}"
            fi
            if [[ "$yaml_line" =~ ^[[:space:]]*name:[[:space:]]*([A-Za-z0-9_-]+) ]]; then
                resource_name="${BASH_REMATCH[1]}"
            fi
        done <<< "$yaml_content"
        
        # Add check if we found both kind and name
        if [[ -n "$resource_kind" && -n "$resource_name" ]]; then
            echo "echo \"Running: Create $resource_kind $resource_name if it doesn't exist\"" >> $OUTPUT_FILE
            echo "# Add a small delay to allow resources to propagate" >> $OUTPUT_FILE
            echo "sleep 3" >> $OUTPUT_FILE
            echo "if ! kubectl get $resource_kind $resource_name &>/dev/null; then" >> $OUTPUT_FILE
            echo "  echo \"Creating $resource_kind $resource_name...\"" >> $OUTPUT_FILE
            echo "  echo '$yaml_content' | kubectl apply -f -" >> $OUTPUT_FILE
            
            # Add wait logic based on resource kind
            if [[ "$resource_kind" == "LLM" ]]; then
                echo "  echo \"Waiting for $resource_kind $resource_name to become ready (up to 20 seconds)...\"" >> $OUTPUT_FILE
                echo "  for i in {1..10}; do" >> $OUTPUT_FILE
                echo "    if kubectl get llm $resource_name -o jsonpath='{.status.ready}' 2>/dev/null | grep -q 'true'; then" >> $OUTPUT_FILE
                echo "      echo \"$resource_kind $resource_name is ready!\"" >> $OUTPUT_FILE
                echo "      break" >> $OUTPUT_FILE
                echo "    fi" >> $OUTPUT_FILE
                echo "    sleep 2" >> $OUTPUT_FILE
                echo "    echo -n \".\"" >> $OUTPUT_FILE
                echo "  done" >> $OUTPUT_FILE
                echo "  echo \"\"" >> $OUTPUT_FILE
            elif [[ "$resource_kind" == "Agent" ]]; then
                echo "  echo \"Waiting for $resource_kind $resource_name to become ready (up to 20 seconds)...\"" >> $OUTPUT_FILE
                echo "  for i in {1..10}; do" >> $OUTPUT_FILE
                echo "    if kubectl get agent $resource_name -o jsonpath='{.status.ready}' 2>/dev/null | grep -q 'true'; then" >> $OUTPUT_FILE
                echo "      echo \"$resource_kind $resource_name is ready!\"" >> $OUTPUT_FILE
                echo "      break" >> $OUTPUT_FILE
                echo "    fi" >> $OUTPUT_FILE
                echo "    sleep 2" >> $OUTPUT_FILE
                echo "    echo -n \".\"" >> $OUTPUT_FILE
                echo "  done" >> $OUTPUT_FILE
                echo "  echo \"\"" >> $OUTPUT_FILE
            elif [[ "$resource_kind" == "MCPServer" ]]; then
                echo "  echo \"Waiting for $resource_kind $resource_name to become ready (up to 30 seconds)...\"" >> $OUTPUT_FILE
                echo "  for i in {1..15}; do" >> $OUTPUT_FILE
                echo "    if kubectl get mcpserver $resource_name -o jsonpath='{.status.ready}' 2>/dev/null | grep -q 'true'; then" >> $OUTPUT_FILE
                echo "      echo \"$resource_kind $resource_name is ready!\"" >> $OUTPUT_FILE
                echo "      break" >> $OUTPUT_FILE
                echo "    fi" >> $OUTPUT_FILE
                echo "    sleep 2" >> $OUTPUT_FILE
                echo "    echo -n \".\"" >> $OUTPUT_FILE
                echo "  done" >> $OUTPUT_FILE
                echo "  echo \"\"" >> $OUTPUT_FILE
            elif [[ "$resource_kind" == "Task" ]]; then
                echo "  echo \"Waiting for $resource_kind $resource_name to complete (up to 60 seconds)...\"" >> $OUTPUT_FILE
                echo "  for i in {1..30}; do" >> $OUTPUT_FILE
                echo "    status=\$(kubectl get task $resource_name -o jsonpath='{.status.phase}' 2>/dev/null || echo \"Pending\")" >> $OUTPUT_FILE
                echo "    if [[ \"\$status\" == \"FinalAnswer\" ]]; then" >> $OUTPUT_FILE
                echo "      echo \"$resource_kind $resource_name completed successfully!\"" >> $OUTPUT_FILE
                echo "      kubectl get task $resource_name -o jsonpath='{.status.output}'" >> $OUTPUT_FILE
                echo "      echo \"\"" >> $OUTPUT_FILE
                echo "      break" >> $OUTPUT_FILE
                echo "    fi" >> $OUTPUT_FILE
                echo "    sleep 2" >> $OUTPUT_FILE
                echo "    echo -n \".\"" >> $OUTPUT_FILE
                echo "  done" >> $OUTPUT_FILE
                echo "  echo \"\"" >> $OUTPUT_FILE
            fi
            
            echo "else" >> $OUTPUT_FILE
            echo "  echo \"$resource_kind $resource_name already exists, updating it...\"" >> $OUTPUT_FILE
            echo "  echo '$yaml_content' | kubectl apply -f -" >> $OUTPUT_FILE
            echo "fi" >> $OUTPUT_FILE
        else
            # If we couldn't determine the resource type/name, just apply it
            echo "echo \"Running: kubectl apply for YAML resource\"" >> $OUTPUT_FILE
            echo "echo '$yaml_content' | kubectl apply -f -" >> $OUTPUT_FILE
        fi
        echo "continue_prompt" >> $OUTPUT_FILE
        echo "" >> $OUTPUT_FILE
        return 0
    fi
    return 1
}

# Extract kubectl commands
extract_kubectl_commands() {
    local line="$1"
    # Skip specific version URLs that might cause conflicts
    if [[ "$line" =~ kubectl[[:space:]]apply[[:space:]]-f.*v0\. ]]; then
        # Skip versioned URLs - we'll use the latest
        return 0
    fi
    
    # Skip kubectl describe commands which are just for viewing
    if [[ "$line" =~ ^kubectl[[:space:]]describe[[:space:]] ]]; then
        return 0
    fi
    
    # Special handling for the main operator deployment
    if [[ "$line" =~ kubectl[[:space:]]apply[[:space:]]-f.*latest\.yaml ]]; then
        echo "echo \"Running: Deploying ACP controller\"" >> $OUTPUT_FILE
        echo "$line" >> $OUTPUT_FILE
        echo "echo \"Waiting for controller deployment to initialize (30 seconds)...\"" >> $OUTPUT_FILE
        echo "sleep 30" >> $OUTPUT_FILE
        echo "kubectl wait --for=condition=available --timeout=60s deployment/acp-controller-manager || echo \"Controller may still be starting, continuing anyway...\"" >> $OUTPUT_FILE
        echo "continue_prompt" >> $OUTPUT_FILE
        echo "" >> $OUTPUT_FILE
        return 0
    fi
    
    # Special handling for multi-line secret creation commands
    if [[ "$line" =~ ^kubectl[[:space:]]create[[:space:]]secret[[:space:]]generic[[:space:]]([a-z0-9_-]+)[[:space:]]*\\$ ]]; then
        local secret_name="${BASH_REMATCH[1]}"
        # This is a multi-line secret creation command - we need special handling
        echo "echo \"Running: Check if secret $secret_name exists, create if it doesn't\"" >> $OUTPUT_FILE
        echo "if ! kubectl get secret $secret_name &>/dev/null; then" >> $OUTPUT_FILE
        echo "  echo \"Creating secret $secret_name...\"" >> $OUTPUT_FILE
        
        # Handle different secret types based on name
        if [[ "$secret_name" == "openai" ]]; then
            echo "  if [[ -z \"\$OPENAI_API_KEY\" ]]; then" >> $OUTPUT_FILE
            echo "    echo \"Error: OPENAI_API_KEY environment variable is not set. Please set it and try again.\"" >> $OUTPUT_FILE
            echo "    read -p \"Do you want to set it now? (y/n): \" SET_KEY" >> $OUTPUT_FILE
            echo "    if [[ \"\$SET_KEY\" == \"y\" ]]; then" >> $OUTPUT_FILE
            echo "      read -p \"Enter your OpenAI API key: \" OPENAI_API_KEY" >> $OUTPUT_FILE
            echo "      export OPENAI_API_KEY" >> $OUTPUT_FILE
            echo "    else" >> $OUTPUT_FILE
            echo "      exit 1" >> $OUTPUT_FILE
            echo "    fi" >> $OUTPUT_FILE
            echo "  fi" >> $OUTPUT_FILE
            echo "  echo \"Creating OpenAI secret with your API key...\"" >> $OUTPUT_FILE
            echo "  kubectl create secret generic $secret_name --from-literal=OPENAI_API_KEY=\$OPENAI_API_KEY --namespace=default" >> $OUTPUT_FILE
        elif [[ "$secret_name" == "anthropic" ]]; then
            echo "  if [[ -z \"\$ANTHROPIC_API_KEY\" ]]; then" >> $OUTPUT_FILE
            echo "    echo \"Error: ANTHROPIC_API_KEY environment variable is not set. Please set it and try again.\"" >> $OUTPUT_FILE
            echo "    read -p \"Do you want to set it now? (y/n): \" SET_KEY" >> $OUTPUT_FILE
            echo "    if [[ \"\$SET_KEY\" == \"y\" ]]; then" >> $OUTPUT_FILE
            echo "      read -p \"Enter your Anthropic API key: \" ANTHROPIC_API_KEY" >> $OUTPUT_FILE
            echo "      export ANTHROPIC_API_KEY" >> $OUTPUT_FILE
            echo "    else" >> $OUTPUT_FILE
            echo "      echo \"Skipping Anthropic setup\"" >> $OUTPUT_FILE
            echo "      return 0" >> $OUTPUT_FILE
            echo "    fi" >> $OUTPUT_FILE
            echo "  fi" >> $OUTPUT_FILE
            echo "  kubectl create secret generic $secret_name --from-literal=ANTHROPIC_API_KEY=\$ANTHROPIC_API_KEY --namespace=default" >> $OUTPUT_FILE
        elif [[ "$secret_name" == "humanlayer" ]]; then
            echo "  if [[ -z \"\$HUMANLAYER_API_KEY\" ]]; then" >> $OUTPUT_FILE
            echo "    echo \"Error: HUMANLAYER_API_KEY environment variable is not set. Please set it and try again.\"" >> $OUTPUT_FILE
            echo "    read -p \"Do you want to set it now? (y/n): \" SET_KEY" >> $OUTPUT_FILE
            echo "    if [[ \"\$SET_KEY\" == \"y\" ]]; then" >> $OUTPUT_FILE
            echo "      read -p \"Enter your HumanLayer API key: \" HUMANLAYER_API_KEY" >> $OUTPUT_FILE
            echo "      export HUMANLAYER_API_KEY" >> $OUTPUT_FILE
            echo "    else" >> $OUTPUT_FILE
            echo "      echo \"Skipping HumanLayer setup\"" >> $OUTPUT_FILE
            echo "      return 0" >> $OUTPUT_FILE
            echo "    fi" >> $OUTPUT_FILE
            echo "  fi" >> $OUTPUT_FILE
            echo "  kubectl create secret generic $secret_name --from-literal=HUMANLAYER_API_KEY=\$HUMANLAYER_API_KEY --namespace=default" >> $OUTPUT_FILE
        else
            # Generic secret handling
            echo "  # Generic secret creation" >> $OUTPUT_FILE
            echo "  $line" >> $OUTPUT_FILE
        fi
        
        echo "  echo \"Secret $secret_name created successfully\"" >> $OUTPUT_FILE
        echo "  kubectl get secret $secret_name" >> $OUTPUT_FILE
        echo "else" >> $OUTPUT_FILE
        echo "  echo \"Secret $secret_name already exists, skipping creation\"" >> $OUTPUT_FILE
        echo "fi" >> $OUTPUT_FILE
        echo "continue_prompt" >> $OUTPUT_FILE
        echo "" >> $OUTPUT_FILE
        return 0
    elif [[ "$line" =~ ^kubectl[[:space:]]create[[:space:]]secret[[:space:]]generic[[:space:]]([a-z0-9_-]+)[[:space:]] ]]; then
        # Single line secret creation
        local secret_name="${BASH_REMATCH[1]}"
        echo "echo \"Running: Check if secret $secret_name exists, create if it doesn't\"" >> $OUTPUT_FILE
        echo "if ! kubectl get secret $secret_name &>/dev/null; then" >> $OUTPUT_FILE
        echo "  echo \"Creating secret $secret_name...\"" >> $OUTPUT_FILE
        echo "  $line" >> $OUTPUT_FILE
        echo "else" >> $OUTPUT_FILE
        echo "  echo \"Secret $secret_name already exists, skipping creation\"" >> $OUTPUT_FILE
        echo "fi" >> $OUTPUT_FILE
        echo "continue_prompt" >> $OUTPUT_FILE
        echo "" >> $OUTPUT_FILE
        return 0
    fi
    
    # Handle kubectl get commands
    if [[ "$line" =~ ^kubectl[[:space:]]get[[:space:]]([a-z]+)[[:space:]]?([a-z0-9_-]*) ]]; then
        local resource_type="${BASH_REMATCH[1]}"
        local resource_name="${BASH_REMATCH[2]}"
        
        echo "echo \"Running: $line\"" >> $OUTPUT_FILE
        echo "# Add a small delay to allow resources to propagate" >> $OUTPUT_FILE
        echo "sleep 2" >> $OUTPUT_FILE
        echo "$line" >> $OUTPUT_FILE
        echo "continue_prompt" >> $OUTPUT_FILE
        echo "" >> $OUTPUT_FILE
        return 0
    elif [[ "$line" =~ ^kubectl[[:space:]]apply[[:space:]]-f.*$ ]]; then
        # Just echo and run kubectl apply commands
        echo "echo \"Running: $line\"" >> $OUTPUT_FILE
        echo "$line" >> $OUTPUT_FILE
        echo "continue_prompt" >> $OUTPUT_FILE
        echo "" >> $OUTPUT_FILE
        return 0
    elif [[ "$line" =~ ^kubectl[[:space:]]([a-z]+)[[:space:]]([a-z0-9-]+).*$ ]]; then
        echo "echo \"Running: $line\"" >> $OUTPUT_FILE
        echo "$line" >> $OUTPUT_FILE
        echo "continue_prompt" >> $OUTPUT_FILE
        echo "" >> $OUTPUT_FILE
        return 0
    fi
    return 1
}

# Extract export commands
extract_export_commands() {
    local line="$1"
    if [[ "$line" =~ ^export[[:space:]]([A-Z_]+)=.*$ ]]; then
        echo "echo \"Running: $line\"" >> $OUTPUT_FILE
        echo "$line" >> $OUTPUT_FILE
        echo "continue_prompt" >> $OUTPUT_FILE
        echo "" >> $OUTPUT_FILE
        return 0
    fi
    return 1
}

# Extract kind commands
extract_kind_commands() {
    local line="$1"
    if [[ "$line" =~ ^kind[[:space:]]create[[:space:]]cluster.*$ ]]; then
        # Add safety check for creating a kind cluster
        echo "echo \"Running: Check if kind cluster exists, create if it doesn't\"" >> $OUTPUT_FILE
        echo "if ! kind get clusters 2>/dev/null | grep -q \"^kind$\"; then" >> $OUTPUT_FILE
        echo "  echo \"Creating new kind cluster...\"" >> $OUTPUT_FILE
        echo "  $line" >> $OUTPUT_FILE
        echo "else" >> $OUTPUT_FILE
        echo "  echo \"Kind cluster already exists, using existing cluster\"" >> $OUTPUT_FILE
        echo "fi" >> $OUTPUT_FILE
        echo "continue_prompt" >> $OUTPUT_FILE
        echo "" >> $OUTPUT_FILE
        return 0
    elif [[ "$line" =~ ^kind[[:space:]]([a-z]+)[[:space:]].*$ ]]; then
        echo "echo \"Running: $line\"" >> $OUTPUT_FILE
        echo "$line" >> $OUTPUT_FILE
        echo "continue_prompt" >> $OUTPUT_FILE
        echo "" >> $OUTPUT_FILE
        return 0
    fi
    return 1
}

# Second pass to catch specific command patterns
while IFS= read -r line; do
    # Skip comment lines
    if [[ "$line" =~ ^#.*$ ]]; then
        continue
    fi
    
    extract_echo_blocks "$line" || extract_kubectl_commands "$line" || extract_export_commands "$line" || extract_kind_commands "$line"
done < "$README_PATH"

# First add the setup banner at the beginning of the script
TMP_FILE=$(mktemp)
cat > $TMP_FILE << 'EOF'
#!/bin/bash

# Commands extracted from ./README.md
# Generated on TIMESTAMP

# Set -e to exit on error
set -e

# Add a function to check if we should continue after each step
continue_prompt() {
  read -p "Press Enter to continue to the next command, or Ctrl+C to exit..." dummy
  echo ""
}

# Banner information
cat << 'BANNER'
====================================================
  ACP (Agent Control Plane) Setup Script
  Generated from README.md on TIMESTAMP
  
  This script will guide you through setting up ACP
  Press Ctrl+C at any time to exit
====================================================

Before continuing, please make sure:
  - You have kubectl installed
  - You have kind installed
  - Docker is running
  - You have your OpenAI API key ready (or set as OPENAI_API_KEY)
BANNER

# Check for required tools
if ! command -v kubectl &> /dev/null; then
  echo "Error: kubectl is not installed. Please install it and try again."
  exit 1
fi

if ! command -v kind &> /dev/null; then
  echo "Error: kind is not installed. Please install it and try again."
  exit 1
fi

# Check if Docker is running
if ! docker info &>/dev/null; then
  echo "Error: Docker is not running. Please start Docker and try again."
  exit 1
fi

# Check for OPENAI_API_KEY
if [[ -z "$OPENAI_API_KEY" ]]; then
  echo "Warning: OPENAI_API_KEY environment variable is not set."
  read -p "Do you want to set it now? (y/n): " SET_KEY
  if [[ "$SET_KEY" == "y" ]]; then
    read -p "Enter your OpenAI API key: " OPENAI_API_KEY
    export OPENAI_API_KEY
  else
    echo "Cannot proceed without an OpenAI API key."
    exit 1
  fi
else
  echo "✅ OPENAI_API_KEY environment variable is set."
fi

read -p "Press Enter to begin setup or Ctrl+C to exit..." dummy
echo ""
EOF

# Replace timestamp
sed "s/TIMESTAMP/$(date)/" $TMP_FILE > $OUTPUT_FILE
rm $TMP_FILE

# At the end of the process, add the commands in the right order
echo -e "\n# Checking if essential resources were created" >> $OUTPUT_FILE
echo "echo \"Checking if essential ACP resources were created...\"" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE

# LLM creation
echo "# Create the LLM resource" >> $OUTPUT_FILE
echo "echo \"Setting up LLM resource for GPT-4o...\"" >> $OUTPUT_FILE
echo "if ! kubectl get llm gpt-4o &>/dev/null; then" >> $OUTPUT_FILE
echo "  echo 'apiVersion: acp.humanlayer.dev/v1alpha1 " >> $OUTPUT_FILE
echo "kind: LLM" >> $OUTPUT_FILE
echo "metadata:" >> $OUTPUT_FILE
echo "  name: gpt-4o" >> $OUTPUT_FILE
echo "spec:" >> $OUTPUT_FILE
echo "  provider: openai" >> $OUTPUT_FILE
echo "  parameters:" >> $OUTPUT_FILE
echo "    model: gpt-4o" >> $OUTPUT_FILE
echo "  apiKeyFrom:" >> $OUTPUT_FILE
echo "    secretKeyRef:" >> $OUTPUT_FILE
echo "      name: openai" >> $OUTPUT_FILE
echo "      key: OPENAI_API_KEY" >> $OUTPUT_FILE
echo "' | kubectl apply -f -" >> $OUTPUT_FILE
echo "  echo \"Waiting for LLM to initialize...\"" >> $OUTPUT_FILE
echo "  for i in {1..10}; do" >> $OUTPUT_FILE
echo "    if kubectl get llm gpt-4o -o jsonpath='{.status.ready}' 2>/dev/null | grep -q 'true'; then" >> $OUTPUT_FILE
echo "      echo \"LLM gpt-4o is ready!\"" >> $OUTPUT_FILE
echo "      break" >> $OUTPUT_FILE
echo "    fi" >> $OUTPUT_FILE
echo "    sleep 2" >> $OUTPUT_FILE
echo "    echo -n \".\"" >> $OUTPUT_FILE
echo "  done" >> $OUTPUT_FILE
echo "  echo \"\"" >> $OUTPUT_FILE
echo "else" >> $OUTPUT_FILE
echo "  echo \"LLM gpt-4o already exists\"" >> $OUTPUT_FILE
echo "fi" >> $OUTPUT_FILE
echo "continue_prompt" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE

# Agent creation
echo "# Create the Agent resource" >> $OUTPUT_FILE
echo "echo \"Creating Agent resource...\"" >> $OUTPUT_FILE
echo "if ! kubectl get agent my-assistant &>/dev/null; then" >> $OUTPUT_FILE
echo "  echo 'apiVersion: acp.humanlayer.dev/v1alpha1 " >> $OUTPUT_FILE
echo "kind: Agent" >> $OUTPUT_FILE
echo "metadata:" >> $OUTPUT_FILE
echo "  name: my-assistant" >> $OUTPUT_FILE
echo "spec:" >> $OUTPUT_FILE
echo "  llmRef:" >> $OUTPUT_FILE
echo "    name: gpt-4o" >> $OUTPUT_FILE
echo "  system: |" >> $OUTPUT_FILE
echo "    You are a helpful assistant. Your job is to help the user with their tasks." >> $OUTPUT_FILE
echo "' | kubectl apply -f -" >> $OUTPUT_FILE
echo "  echo \"Waiting for Agent to initialize...\"" >> $OUTPUT_FILE
echo "  for i in {1..10}; do" >> $OUTPUT_FILE
echo "    if kubectl get agent my-assistant -o jsonpath='{.status.ready}' 2>/dev/null | grep -q 'true'; then" >> $OUTPUT_FILE
echo "      echo \"Agent my-assistant is ready!\"" >> $OUTPUT_FILE
echo "      break" >> $OUTPUT_FILE
echo "    fi" >> $OUTPUT_FILE
echo "    sleep 2" >> $OUTPUT_FILE
echo "    echo -n \".\"" >> $OUTPUT_FILE
echo "  done" >> $OUTPUT_FILE
echo "  echo \"\"" >> $OUTPUT_FILE
echo "else" >> $OUTPUT_FILE
echo "  echo \"Agent my-assistant already exists\"" >> $OUTPUT_FILE
echo "fi" >> $OUTPUT_FILE
echo "continue_prompt" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE

# First task creation - hello-world
echo "# Create a task to interact with the agent" >> $OUTPUT_FILE
echo "echo \"Creating a task to interact with your agent...\"" >> $OUTPUT_FILE
echo "if ! kubectl get task hello-world-1 &>/dev/null; then" >> $OUTPUT_FILE
echo "  echo 'apiVersion: acp.humanlayer.dev/v1alpha1 " >> $OUTPUT_FILE
echo "kind: Task" >> $OUTPUT_FILE
echo "metadata:" >> $OUTPUT_FILE
echo "  name: hello-world-1" >> $OUTPUT_FILE
echo "spec:" >> $OUTPUT_FILE
echo "  agentRef:" >> $OUTPUT_FILE
echo "    name: my-assistant" >> $OUTPUT_FILE
echo "  userMessage: \"What is the capital of the moon?\"" >> $OUTPUT_FILE
echo "' | kubectl apply -f -" >> $OUTPUT_FILE
echo "  echo \"Waiting for Task to complete...\"" >> $OUTPUT_FILE
echo "  for i in {1..15}; do" >> $OUTPUT_FILE
echo "    status=\$(kubectl get task hello-world-1 -o jsonpath='{.status.phase}' 2>/dev/null || echo \"Pending\")" >> $OUTPUT_FILE
echo "    if [[ \"\$status\" == \"FinalAnswer\" ]]; then" >> $OUTPUT_FILE
echo "      echo \"Task hello-world-1 completed successfully!\"" >> $OUTPUT_FILE
echo "      echo \"Result:\"" >> $OUTPUT_FILE
echo "      kubectl get task hello-world-1 -o jsonpath='{.status.output}'" >> $OUTPUT_FILE
echo "      echo \"\"" >> $OUTPUT_FILE
echo "      break" >> $OUTPUT_FILE
echo "    fi" >> $OUTPUT_FILE
echo "    sleep 2" >> $OUTPUT_FILE
echo "    echo -n \".\"" >> $OUTPUT_FILE
echo "  done" >> $OUTPUT_FILE
echo "  echo \"\"" >> $OUTPUT_FILE
echo "else" >> $OUTPUT_FILE
echo "  echo \"Task hello-world-1 already exists\"" >> $OUTPUT_FILE
echo "fi" >> $OUTPUT_FILE
echo "continue_prompt" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE

# MCP server setup
echo "# Add MCP server setup" >> $OUTPUT_FILE
echo "echo \"Setting up MCP server for fetch tool...\"" >> $OUTPUT_FILE
echo "if ! kubectl get mcpserver fetch &>/dev/null; then" >> $OUTPUT_FILE
echo "  echo 'apiVersion: acp.humanlayer.dev/v1alpha1 " >> $OUTPUT_FILE
echo "kind: MCPServer" >> $OUTPUT_FILE
echo "metadata:" >> $OUTPUT_FILE
echo "  name: fetch" >> $OUTPUT_FILE
echo "spec:" >> $OUTPUT_FILE
echo "  transport: \"stdio\"" >> $OUTPUT_FILE
echo "  command: \"uvx\"" >> $OUTPUT_FILE
echo "  args: [\"mcp-server-fetch\"]" >> $OUTPUT_FILE
echo "' | kubectl apply -f -" >> $OUTPUT_FILE
echo "  echo \"Waiting for MCPServer fetch to initialize...\"" >> $OUTPUT_FILE
echo "  for i in {1..10}; do" >> $OUTPUT_FILE
echo "    if kubectl get mcpserver fetch -o jsonpath='{.status.ready}' 2>/dev/null | grep -q 'true'; then" >> $OUTPUT_FILE
echo "      echo \"MCPServer fetch is ready!\"" >> $OUTPUT_FILE
echo "      break" >> $OUTPUT_FILE
echo "    fi" >> $OUTPUT_FILE
echo "    sleep 2" >> $OUTPUT_FILE
echo "    echo -n \".\"" >> $OUTPUT_FILE
echo "  done" >> $OUTPUT_FILE
echo "  echo \"\"" >> $OUTPUT_FILE
echo "else" >> $OUTPUT_FILE
echo "  echo \"MCPServer fetch already exists\"" >> $OUTPUT_FILE
echo "fi" >> $OUTPUT_FILE
echo "continue_prompt" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE

# Update agent to use the fetch tool
echo "# Update agent to use fetch tool" >> $OUTPUT_FILE
echo "echo \"Updating agent to use fetch tool...\"" >> $OUTPUT_FILE
echo "echo 'apiVersion: acp.humanlayer.dev/v1alpha1 " >> $OUTPUT_FILE
echo "kind: Agent" >> $OUTPUT_FILE
echo "metadata:" >> $OUTPUT_FILE
echo "  name: my-assistant" >> $OUTPUT_FILE
echo "spec:" >> $OUTPUT_FILE
echo "  llmRef:" >> $OUTPUT_FILE
echo "    name: gpt-4o" >> $OUTPUT_FILE
echo "  system: |" >> $OUTPUT_FILE
echo "    You are a helpful assistant. Your job is to help the user with their tasks." >> $OUTPUT_FILE
echo "  mcpServers:" >> $OUTPUT_FILE
echo "    - name: fetch" >> $OUTPUT_FILE
echo "' | kubectl apply -f -" >> $OUTPUT_FILE
echo "echo \"Waiting for updated agent to initialize...\"" >> $OUTPUT_FILE
echo "sleep 5" >> $OUTPUT_FILE
echo "kubectl get agent my-assistant -o wide" >> $OUTPUT_FILE
echo "continue_prompt" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE

# Create a task that uses the fetch tool
echo "# Create a task that uses the fetch tool" >> $OUTPUT_FILE
echo "echo \"Creating a task that uses the fetch tool...\"" >> $OUTPUT_FILE
echo "if ! kubectl get task fetch-task &>/dev/null; then" >> $OUTPUT_FILE
echo "  echo 'apiVersion: acp.humanlayer.dev/v1alpha1 " >> $OUTPUT_FILE
echo "kind: Task" >> $OUTPUT_FILE
echo "metadata:" >> $OUTPUT_FILE
echo "  name: fetch-task" >> $OUTPUT_FILE
echo "spec:" >> $OUTPUT_FILE
echo "  agentRef:" >> $OUTPUT_FILE
echo "    name: my-assistant" >> $OUTPUT_FILE
echo "  userMessage: \"what is the data at https://lotrapi.co/api/v1/characters/1?\"" >> $OUTPUT_FILE
echo "' | kubectl apply -f -" >> $OUTPUT_FILE
echo "  echo \"Waiting for fetch-task to complete...\"" >> $OUTPUT_FILE
echo "  for i in {1..30}; do" >> $OUTPUT_FILE
echo "    status=\$(kubectl get task fetch-task -o jsonpath='{.status.phase}' 2>/dev/null || echo \"Pending\")" >> $OUTPUT_FILE
echo "    if [[ \"\$status\" == \"FinalAnswer\" ]]; then" >> $OUTPUT_FILE
echo "      echo \"Task fetch-task completed successfully!\"" >> $OUTPUT_FILE
echo "      echo \"Result:\"" >> $OUTPUT_FILE
echo "      kubectl get task fetch-task -o jsonpath='{.status.output}'" >> $OUTPUT_FILE
echo "      echo \"\"" >> $OUTPUT_FILE
echo "      break" >> $OUTPUT_FILE
echo "    fi" >> $OUTPUT_FILE
echo "    sleep 2" >> $OUTPUT_FILE
echo "    echo -n \".\"" >> $OUTPUT_FILE
echo "  done" >> $OUTPUT_FILE
echo "  echo \"\"" >> $OUTPUT_FILE
echo "else" >> $OUTPUT_FILE
echo "  echo \"Task fetch-task already exists\"" >> $OUTPUT_FILE
echo "fi" >> $OUTPUT_FILE
echo "continue_prompt" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE


# Add a final message
echo "# Add completion message" >> $OUTPUT_FILE
echo "cat << 'EOF'" >> $OUTPUT_FILE
echo "====================================================" >> $OUTPUT_FILE
echo "  ACP Setup Complete!" >> $OUTPUT_FILE
echo "  " >> $OUTPUT_FILE
echo "  You can now interact with ACP using kubectl:" >> $OUTPUT_FILE
echo "  - kubectl get llm" >> $OUTPUT_FILE
echo "  - kubectl get agent" >> $OUTPUT_FILE
echo "  - kubectl get task" >> $OUTPUT_FILE
echo "  - kubectl get mcpserver" >> $OUTPUT_FILE
echo "  " >> $OUTPUT_FILE
echo "  When you're done, you can clean up with:" >> $OUTPUT_FILE
echo "  - kubectl delete toolcall --all" >> $OUTPUT_FILE
echo "  - kubectl delete task --all" >> $OUTPUT_FILE
echo "  - kubectl delete agent --all" >> $OUTPUT_FILE
echo "  - kubectl delete mcpserver --all" >> $OUTPUT_FILE
echo "  - kubectl delete contactchannel --all" >> $OUTPUT_FILE
echo "  - kubectl delete llm --all" >> $OUTPUT_FILE
echo "  - kubectl delete secret openai anthropic humanlayer" >> $OUTPUT_FILE
echo "  - kind delete cluster" >> $OUTPUT_FILE
echo "====================================================" >> $OUTPUT_FILE
echo "EOF" >> $OUTPUT_FILE

# Make the script executable
chmod +x $OUTPUT_FILE

echo "Commands have been extracted to $OUTPUT_FILE"
echo "Review the file contents before running:"
echo "--------------------------------------"
cat $OUTPUT_FILE
echo "--------------------------------------"
echo "To run the commands, execute: $OUTPUT_FILE"