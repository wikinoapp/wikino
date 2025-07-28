# typed: false
# frozen_string_literal: true

RSpec.configure do |config|
  config.around do |example|
    I18n.with_locale(:ja) do
      example.run
    end
  end
end
