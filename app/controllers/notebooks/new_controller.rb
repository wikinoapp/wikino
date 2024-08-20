# typed: true
# frozen_string_literal: true

module Notebooks
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable

    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      @form = NewNotebookForm.new
    end
  end
end
