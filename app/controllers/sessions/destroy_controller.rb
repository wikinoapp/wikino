# typed: true
# frozen_string_literal: true

module Sessions
  class DestroyController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :set_current_space
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      sign_out

      redirect_to root_path
    end
  end
end
