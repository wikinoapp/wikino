# typed: false
# frozen_string_literal: true

require "view_component/test_helpers"

RSpec.configure do |config|
  config.include ViewComponent::TestHelpers, type: :view
  config.include Capybara::RSpecMatchers, type: :view

  config.around(type: :view) do |example|
    I18n.with_locale(:ja) do
      example.run
    end
  end
end
