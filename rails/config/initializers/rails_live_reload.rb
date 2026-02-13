# typed: false
# frozen_string_literal: true

if defined?(RailsLiveReload)
  RailsLiveReload.configure do |config|
    # config.url = "/rails/live/reload"

    config.watch %r{app/views/.+\.(erb|rb)$}, reload: :always
    config.watch %r{app/components/.+\.(erb|rb)$}, reload: :always
    config.watch %r{(app|vendor)/(assets|javascript)/\w+/(.+\.(css|js|html|png|jpg|ts)).*}, reload: :always
    config.watch %r{config/locales/.+\.yml}, reload: :always

    # More examples:
    # config.watch %r{app/helpers/.+\.rb}, reload: :always

    # config.enabled = Rails.env.development?
  end
end
