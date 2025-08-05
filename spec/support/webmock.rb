# typed: false
# frozen_string_literal: true

require "webmock/rspec"

RSpec.configure do |config|
  # System specではWebMockを完全に無効化
  config.before(:each, type: :system) do
    WebMock.allow_net_connect!
  end

  # System spec以外でWebMockを有効化
  config.before(:each) do |example|
    unless example.metadata[:type] == :system
      WebMock.disable_net_connect!(
        allow_localhost: true,
        allow: [
          "chromedriver.storage.googleapis.com",
          "127.0.0.1",
          "0.0.0.0",
          "localhost"
        ]
      )
    end
  end

  # 各テスト後にWebMockをリセット
  config.after(:each) do
    WebMock.reset!
  end
end