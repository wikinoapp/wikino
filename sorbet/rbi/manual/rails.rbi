# typed: strict
# frozen_string_literal: true

class Rails::Application::Configuration < ::Rails::Engine::Configuration
  def action_controller; end
  def action_dispatch; end
  def action_mailer; end
  def active_record; end
  def active_support; end
  def i18n; end
end
