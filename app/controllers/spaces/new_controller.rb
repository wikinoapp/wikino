# typed: true
# frozen_string_literal: true

module Spaces
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      form = Spaces::CreationForm.new

      render_component Spaces::NewView.new(
        current_user: current_user!,
        form:
      )
    end
  end
end
