# Custom CRUD Provider Examples

This directory contains examples that demonstrate how to use the Custom CRUD Provider.

## File Management Example

The `main.tf` file demonstrates a basic example of using the provider to manage a file through custom scripts. The example:

1. Creates a file with specified content
2. Reads the file's content
3. Updates the file content when changed
4. Deletes the file when the resource is destroyed

### Usage

1. Ensure the provider is built and installed
2. Make the CRUD scripts executable:
   ```bash
   chmod +x crud/*.sh
   ```
3. Initialize Terraform:
   ```bash
   terraform init
   ```
4. Apply the configuration:
   ```bash
   terraform apply
   ```

### Script Details

The example uses four scripts in the `crud` directory:
- `create.sh`: Creates a new file with the specified content
- `read.sh`: Reads the current content of the file
- `update.sh`: Updates the file with new content
- `delete.sh`: Removes the file

Each script follows the required JSON input/output format and handles appropriate error conditions.
