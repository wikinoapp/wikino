# typed: true
# frozen_string_literal: true

module Lists
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable

    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      @form = NewListForm.new
    end
  end
end
