# typed: true
# frozen_string_literal: true

module Topics
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::Authorizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      @form = NewTopicForm.new
    end
  end
end
