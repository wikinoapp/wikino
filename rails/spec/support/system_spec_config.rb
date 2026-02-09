# typed: false
# frozen_string_literal: true

module SystemSpecHelpers
  extend T::Sig

  def sign_in(user_record:, password: "passw0rd")
    visit "/_test/sign_in?user_id=#{user_record.id}"
    expect(page).to have_current_path("/home")
  end
end

RSpec.configure do |config|
  config.include SystemSpecHelpers, type: :system
end
