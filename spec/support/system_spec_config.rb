# typed: false
# frozen_string_literal: true

module SystemSpecHelpers
  extend T::Sig

  def sign_in(user_record:, password: "passw0rd")
    visit "/sign_in"

    # Debug: let's check what form fields are actually present
    within("form.form") do
      # Find email and password fields by their type
      email_field = find('input[type="email"]')
      password_field = find('input[type="password"]')

      # Fill in the fields
      email_field.set(user_record.email)
      password_field.set(password)

      # Submit the form
      find('button[type="submit"]').click
    end

    # Wait for navigation or check for errors
    Capybara.using_wait_time(5) do
      # Check if we're still on sign in page after a moment
      sleep 0.5
      if page.current_path == "/sign_in"
        # Check for error message
        if page.has_css?('[role="alert"]', wait: 1)
          error_text = find('[role="alert"]').text
          raise "Sign in failed with error: #{error_text}"
        else
          # Debug: save page HTML
          save_page
          raise "Sign in failed - still on sign in page with no error message"
        end
      end

      expect(page).to have_current_path("/home")
    end
  end
end

RSpec.configure do |config|
  config.include SystemSpecHelpers, type: :system
end
