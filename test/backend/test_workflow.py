# test/backend/test_workflow.py
import requests
import json
import os
import time
from PIL import Image

BASE_URL = "http://localhost:8080/api"
ADMIN_USERNAME = "admin"
ADMIN_PASSWORD = "verysecret"
TEST_USERNAME = "testuser"
TEST_PASSWORD = "testpassword"
IMAGE_DATABASE_NAME = "test_image_db"
AUDIO_DATABASE_NAME = "test_audio_db"
FILE_DATABASE_NAME = "test_file_db"
IMAGE_ID = None
AUDIO_ID = None
FILE_ID = None
DUMMY_IMAGE_PATH = "dummy.png"
DUMMY_AUDIO_PATH = "dummy.mp3"
DUMMY_FILE_PATH = "dummy.txt"

def create_dummy_image():
    """
    Creates a dummy PNG image.
    """
    img = Image.new('RGB', (100, 100), color = 'red')
    img.save(DUMMY_IMAGE_PATH, 'PNG')

def create_dummy_audio():
    """
    Creates a dummy audio file.
    """
    with open(DUMMY_AUDIO_PATH, "wb") as f:
        f.write(b"dummy mp3 data")

def create_dummy_file():
    """
    Creates a dummy file.
    """
    with open(DUMMY_FILE_PATH, "wb") as f:
        f.write(b"dummy file data")

def cleanup():
    """
    Cleans up created resources.
    """
    # Use admin credentials for cleanup
    auth = (ADMIN_USERNAME, ADMIN_PASSWORD)

    if IMAGE_ID:
        print(f"Deleting image {IMAGE_ID}...")
        requests.delete(
            f"{BASE_URL}/entry?database_name={IMAGE_DATABASE_NAME}&id={IMAGE_ID}",
            auth=auth
        )
    if AUDIO_ID:
        print(f"Deleting audio {AUDIO_ID}...")
        requests.delete(
            f"{BASE_URL}/entry?database_name={AUDIO_DATABASE_NAME}&id={AUDIO_ID}",
            auth=auth
        )
    if FILE_ID:
        print(f"Deleting file {FILE_ID}...")
        requests.delete(
            f"{BASE_URL}/entry?database_name={FILE_DATABASE_NAME}&id={FILE_ID}",
            auth=auth
        )

    print(f"Deleting database '{IMAGE_DATABASE_NAME}'...")
    requests.delete(
        f"{BASE_URL}/database?name={IMAGE_DATABASE_NAME}",
        auth=auth
    )
    print(f"Deleting database '{AUDIO_DATABASE_NAME}'...")
    requests.delete(
        f"{BASE_URL}/database?name={AUDIO_DATABASE_NAME}",
        auth=auth
    )
    print(f"Deleting database '{FILE_DATABASE_NAME}'...")
    requests.delete(
        f"{BASE_URL}/database?name={FILE_DATABASE_NAME}",
        auth=auth
    )

    # Clean up test user
    try:
        users_response = requests.get(f"{BASE_URL}/users", auth=auth, timeout=5)
        if users_response.status_code == 200:
            users = users_response.json()
            for user in users:
                if user['username'] == TEST_USERNAME:
                    print(f"Deleting user '{TEST_USERNAME}'...")
                    requests.delete(f"{BASE_URL}/user?id={user['id']}", auth=auth)
                    break
    except requests.RequestException as e:
        print(f"Could not clean up users (server may be down): {e}")


    if os.path.exists(DUMMY_IMAGE_PATH):
        os.remove(DUMMY_IMAGE_PATH)
        print("Dummy image removed.")
    if os.path.exists(DUMMY_AUDIO_PATH):
        os.remove(DUMMY_AUDIO_PATH)
        print("Dummy audio removed.")
    if os.path.exists(DUMMY_FILE_PATH):
        os.remove(DUMMY_FILE_PATH)
        print("Dummy file removed.")

def assert_or_fail(condition, message):
    """Helper to print and exit on assertion failure."""
    if not condition:
        print(f"Assertion FAILED: {message}")
        raise Exception(message) # Let the main try/finally handle cleanup

def wait_for_server(url, timeout=30):
    """Waits for the server to be ready by polling the info endpoint."""
    print("Waiting for server to be ready...")
    start_time = time.time()
    while True:
        try:
            response = requests.get(f"{url}/info", timeout=2)
            if response.status_code == 200:
                print("Server is ready.")
                return True
        except requests.ConnectionError:
            pass # Server not up yet
        
        if time.time() - start_time > timeout:
            print(f"Server did not become ready in {timeout} seconds.")
            return False
        time.sleep(1)

def main():
    """
    Runs a full workflow test of the backend API.
    """
    global IMAGE_ID, AUDIO_ID, FILE_ID

    print("Starting backend workflow test...")

    # Initial cleanup in case of previous failed runs
    cleanup()

    # Create dummy files for testing uploads
    create_dummy_image()
    print("Dummy image created.")
    create_dummy_audio()
    print("Dummy audio created.")
    create_dummy_file()
    print("Dummy file created.")

    try:
        # Wait for the server to be ready
        if not wait_for_server(BASE_URL):
            return

        # User management tests (as admin)
        auth_admin = (ADMIN_USERNAME, ADMIN_PASSWORD)

        # 1. Get all users
        print("Getting all users...")
        response = requests.get(f"{BASE_URL}/users", auth=auth_admin)
        assert_or_fail(response.status_code == 200, f"Error getting users: {response.status_code} {response.text}")
        print("Users retrieved successfully.")

        # 2. Create a new user
        print(f"Creating user '{TEST_USERNAME}'...")
        user_payload = {
            "username": TEST_USERNAME,
            "password": TEST_PASSWORD,
            "can_view": True,
            "can_create": True,
            "can_edit": True,
            "can_delete": True,
            "is_admin": False
        }
        response = requests.post(
            f"{BASE_URL}/user",
            auth=auth_admin,
            json=user_payload
        )
        assert_or_fail(response.status_code == 201, f"Error creating user: {response.status_code} {response.text}")
        test_user_id = response.json()["id"]
        print(f"User '{TEST_USERNAME}' created successfully with ID: {test_user_id}")

        # Database and entry workflow tests (as testuser)
        session = requests.Session()
        session.auth = (TEST_USERNAME, TEST_PASSWORD)

        # 3. Create databases
        print(f"Creating databases as '{TEST_USERNAME}'...")
        db_payloads = [
            {
                "name": IMAGE_DATABASE_NAME,
                "content_type": "image",
                "custom_fields": [{"name": "description", "type": "TEXT"}]
            },
            {
                "name": AUDIO_DATABASE_NAME,
                "content_type": "audio",
                "custom_fields": [{"name": "artist", "type": "TEXT"}]
            },
            {
                "name": FILE_DATABASE_NAME,
                "content_type": "file",
                "custom_fields": [{"name": "description", "type": "TEXT"}]
            }
        ]
        for db_payload in db_payloads:
            response = session.post(
                f"{BASE_URL}/database",
                json=db_payload
            )
            assert_or_fail(response.status_code == 201, f"Error creating database: {response.status_code} {response.text}")
            print(f"Database '{db_payload['name']}' created successfully.")

        # 4. Upload entries
        print(f"Uploading entries as '{TEST_USERNAME}'...")
        # Image
        with open(DUMMY_IMAGE_PATH, "rb") as f:
            data = f.read()
        metadata = {"description": "This is a test image"}
        files = {
            "metadata": (None, json.dumps(metadata), "application/json"),
            "file": (DUMMY_IMAGE_PATH, data, "image/png")
        }
        response = session.post(
            f"{BASE_URL}/entry?database_name={IMAGE_DATABASE_NAME}",
            files=files
        )
        assert_or_fail(response.status_code == 201, f"Error uploading image: {response.status_code} {response.text}")
        IMAGE_ID = response.json()["id"]
        print(f"Image uploaded successfully with ID: {IMAGE_ID}")

        # Audio
        with open(DUMMY_AUDIO_PATH, "rb") as f:
            data = f.read()
        metadata = {"artist": "Test Artist"}
        files = {
            "metadata": (None, json.dumps(metadata), "application/json"),
            "file": (DUMMY_AUDIO_PATH, data, "audio/mpeg")
        }
        response = session.post(
            f"{BASE_URL}/entry?database_name={AUDIO_DATABASE_NAME}",
            files=files
        )
        assert_or_fail(response.status_code == 201, f"Error uploading audio: {response.status_code} {response.text}")
        AUDIO_ID = response.json()["id"]
        print(f"Audio uploaded successfully with ID: {AUDIO_ID}")

        # File
        with open(DUMMY_FILE_PATH, "rb") as f:
            data = f.read()
        metadata = {"description": "This is a test file"}
        files = {
            "metadata": (None, json.dumps(metadata), "application/json"),
            "file": (DUMMY_FILE_PATH, data, "text/plain")
        }
        response = session.post(
            f"{BASE_URL}/entry?database_name={FILE_DATABASE_NAME}",
            files=files
        )
        assert_or_fail(response.status_code == 201, f"Error uploading file: {response.status_code} {response.text}")
        FILE_ID = response.json()["id"]
        print(f"File uploaded successfully with ID: {FILE_ID}")

        # 5. Download and verify entries
        print("Downloading and verifying entries...")
        
        # Verify Image
        response = session.get(f"{BASE_URL}/entry/file?database_name={IMAGE_DATABASE_NAME}&id={IMAGE_ID}")
        assert_or_fail(response.status_code == 200, f"Error downloading image: {response.status_code} {response.text}")
        assert_or_fail(response.headers['Content-Type'] == 'image/png', f"Image Content-Type mismatch: {response.headers['Content-Type']}")
        assert_or_fail(response.headers['Content-Disposition'] == f'attachment; filename="{DUMMY_IMAGE_PATH}"', f"Image Content-Disposition mismatch: {response.headers['Content-Disposition']}")
        with open(DUMMY_IMAGE_PATH, "rb") as f:
            assert_or_fail(response.content == f.read(), "Image content mismatch")
        print("Image entry verified successfully.")

        # Verify Audio
        response = session.get(f"{BASE_URL}/entry/file?database_name={AUDIO_DATABASE_NAME}&id={AUDIO_ID}")
        assert_or_fail(response.status_code == 200, f"Error downloading audio: {response.status_code} {response.text}")
        assert_or_fail(response.headers['Content-Type'] == 'audio/mpeg', f"Audio Content-Type mismatch: {response.headers['Content-Type']}")
        assert_or_fail(response.headers['Content-Disposition'] == f'attachment; filename="{DUMMY_AUDIO_PATH}"', f"Audio Content-Disposition mismatch: {response.headers['Content-Disposition']}")
        with open(DUMMY_AUDIO_PATH, "rb") as f:
            assert_or_fail(response.content == f.read(), "Audio content mismatch")
        print("Audio entry verified successfully.")

        # Verify File
        response = session.get(f"{BASE_URL}/entry/file?database_name={FILE_DATABASE_NAME}&id={FILE_ID}")
        assert_or_fail(response.status_code == 200, f"Error downloading file: {response.status_code} {response.text}")
        assert_or_fail(response.headers['Content-Type'] == 'text/plain', f"File Content-Type mismatch: {response.headers['Content-Type']}")
        assert_or_fail(response.headers['Content-Disposition'] == f'attachment; filename="{DUMMY_FILE_PATH}"', f"File Content-Disposition mismatch: {response.headers['Content-Disposition']}")
        with open(DUMMY_FILE_PATH, "rb") as f:
            assert_or_fail(response.content == f.read(), "File content mismatch")
        print("File entry verified successfully.")

        # 6. Update user roles
        print(f"Updating roles for user '{TEST_USERNAME}'...")
        roles_payload = {
            "can_create": False,
            "can_delete": False
        }
        response = requests.patch(
            f"{BASE_URL}/user?id={test_user_id}",
            auth=auth_admin,
            json=roles_payload
        )
        assert_or_fail(response.status_code == 200 and not response.json()["can_create"], f"Error updating user roles: {response.status_code} {response.text}")
        print("User roles updated successfully.")

        # Re-authenticate the user to get a new session with the updated roles
        session = requests.Session()
        session.auth = (TEST_USERNAME, TEST_PASSWORD)

        # 7. Verify that the user can no longer create a database
        print(f"Verifying that '{TEST_USERNAME}' can no longer create a database...")
        db_payload_2 = {
            "name": "test_db_2",
            "content_type": "image",
            "custom_fields": [{"name": "description", "type": "TEXT"}]
        }
        response = session.post(
            f"{BASE_URL}/database",
            json=db_payload_2
        )
        assert_or_fail(response.status_code == 403, f"Error: User should not be able to create a database, but got status code {response.status_code}")
        print("Successfully verified that the user cannot create a database.")

        # 8. Delete the user
        print(f"Deleting user '{TEST_USERNAME}'...")
        response = requests.delete(f"{BASE_URL}/user?id={test_user_id}", auth=auth_admin)
        assert_or_fail(response.status_code == 200, f"Error deleting user: {response.status_code} {response.text}")
        print("User deleted successfully.")

        print("\nBackend workflow test completed successfully!")

    except Exception as e:
        print(f"\nTest FAILED: {e}")
    finally:
        print("--- Running cleanup ---")
        cleanup()


if __name__ == "__main__":
    main()