# typed: strict
# frozen_string_literal: true

module Spaces
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable

    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
    end
  end
end
