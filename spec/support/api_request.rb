# frozen_string_literal: true

RSpec.configure do |config|
  config.before :each, type: :request do
    host! "api.nonoto.test"
  end
end
