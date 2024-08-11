# typed: true
# frozen_string_literal: true

module Notes
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable

    before_action :require_authentication

    #   sig { returns(T.untyped) }
    #   def call
    #     @note = current_user.notes.new
    #   end
  end
end
