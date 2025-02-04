# typed: strict
# frozen_string_literal: true

module Home
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      active_spaces = T.let(Current.viewer!, User).active_spaces

      render Home::ShowView.new(active_spaces:)
    end
  end
end
