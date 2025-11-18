Please follow the guidelines below:
- all code is written using clean code principles: write small functions, do not repeat yourself, create clean file and folder structures. Add comments to your code.
- write tests for your code, and make sure the tests are passing.
- The Readme.md file is the first thing the user will read. Make sure to review and possibly update it upon making code changes.
- there is no need to update the files in the /docs folder, or the go.mod file, these are generated automatically with CLI tools. ('swag init -g ./cmd/mediahub/main.go' and 'go mod tidy')
- avoid comments like "ADDED" or "REFACTORED", I can see your changes in a Git diff. You can add explanatory comments though.

If you are Gemini and have a Canvas available to display/edit code:
- use the Canvas for code updates or new files
- always recreate the complete files with all the code inside, not just the updates and changes
- all code is written using clean code principles: write small functions, do not repeat yourself, create clean file and folder structures. Add comments to your code.
- write the filepath of each file that you updated or created in a comment in the first line of code
- name the Canvas file according to the filepath as well
- if the response contains too many file changes, you should create the changes in multiple steps, instead of in one large response.