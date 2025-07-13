## file/create.sh"
# input="$(cat)"
# id="$(mktemp)"
# content="$(echo "$input" | jq -r ".input.content")"
# echo "$content" > "$id"
# jq -n --arg id "$id" --arg content "$content" '{id: $id, content: $content}'

## file/read.sh"
# input="$(cat)"
# id="$(echo "$input" | jq -r ".id")"
# content=$(cat "$id") || exit 22
# jq -n --arg id "$id" --arg content "$content" '{id: $id, content: $content}'

## file/update.sh"
# input="$(cat)"
# id="$(echo "$input" | jq -r ".id")"
# content="$(echo "$input" | jq -r ".input.content")"
# echo "$content" > "$id"
# jq -n --arg id "$id" --arg content "$(cat "$id")" '{id: $id, content: $content}'

## file/delete.sh"
# input="$(cat)"
# id="$(echo "$input" | jq -r '.id')"
# rm "$id"

# This example shows how you would create the equivalent of the local_file resource
resource "customcrud" "file" {
  hooks {
    create = "file/create.sh"
    read   = "file/read.sh"
    update = "file/update.sh"
    delete = "file/delete.sh"
  }

  input = {
    content = "Hello, World!"
  }
}
