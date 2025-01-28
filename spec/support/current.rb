# typed: false
# frozen_string_literal: true

RSpec.configure do |config|
  config.after do
    Current.viewer = nil
  end
end
