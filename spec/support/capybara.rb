# typed: false
# frozen_string_literal: true

require "capybara/rails"
require "capybara/rspec"
require "selenium/webdriver"

Capybara.register_driver :chrome_headless do |app|
  options = Selenium::WebDriver::Chrome::Options.new
  options.add_argument("--headless")
  options.add_argument("--window-size=1400,1400")
  options.add_argument("--no-sandbox")
  options.add_argument("--disable-dev-shm-usage")

  if ENV["CI"]
    Capybara::Selenium::Driver.new(
      app,
      browser: :chrome,
      options:
    )
  else
    Capybara::Selenium::Driver.new(
      app,
      browser: :remote,
      url: "http://chrome:4444/wd/hub",
      options:
    )
  end
end

Capybara.configure do |config|
  config.default_driver = :chrome_headless
  config.javascript_driver = :chrome_headless

  config.server = :puma
  config.server_host = "0.0.0.0"
  config.server_port = 4000

  config.app_host = if ENV["CI"]
    "http://localhost:4000"
  else
    "http://#{`hostname`.strip&.downcase}"
  end

  config.default_max_wait_time = 5
  config.disable_animation = true
end

RSpec.configure do |config|
  config.prepend_before(:each, type: :system) do
    driven_by Capybara.javascript_driver
  end

  config.filter_gems_from_backtrace("capybara", "cuprite", "ferrum")
end
