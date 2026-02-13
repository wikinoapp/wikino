# typed: strict
# frozen_string_literal: true

module Settings
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      render_component Settings::ShowView.new(
        current_user: current_user!
      )
    end
  end
end
