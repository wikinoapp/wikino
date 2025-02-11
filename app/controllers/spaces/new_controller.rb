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
      form = NewSpaceForm.new

      render Spaces::NewView.new(form:)
    end
  end
end
