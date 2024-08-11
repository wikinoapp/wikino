# typed: true
# frozen_string_literal: true

module Sessions
  class DestroyController < ApplicationController
    include ControllerConcerns::Authenticatable

    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      sign_out

      redirect_to root_path
    end
  end
end
