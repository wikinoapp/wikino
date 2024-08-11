# typed: true
# frozen_string_literal: true

module Sessions
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable

    before_action :require_no_authentication

    sig { returns(T.untyped) }
    def call
    end
  end
end
