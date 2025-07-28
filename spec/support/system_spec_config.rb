# typed: false
# frozen_string_literal: true

module SystemSpecHelpers
  extend T::Sig

  def sign_in(user_record:, password: "passw0rd")
    visit "/sign_in"

    within("form.form") do
      email_field = find('input[type="email"]')
      password_field = find('input[type="password"]')

      email_field.set(user_record.email)
      password_field.set(password)

      find('button[type="submit"]').click
    end

    expect(page).to have_current_path("/home")
  end
end

RSpec.configure do |config|
  config.include SystemSpecHelpers, type: :system
end
