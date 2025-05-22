# typed: false
# frozen_string_literal: true

module SystemSpecHelpers
  extend T::Sig

  def sign_in(user_record:, password: "passw0rd")
    visit "/sign_in"

    fill_in "Email", with: user_record.email
    fill_in "Password", with: password
    click_button "Sign in"

    expect(page).to have_content "Signed in successfully"
  end
end

RSpec.configure do |config|
  config.include SystemSpecHelpers, type: :system
end
