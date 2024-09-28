# typed: true
# frozen_string_literal: true

module Notebooks
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::NotebookSettable

    around_action :set_locale
    before_action :require_authentication
    before_action :set_notebook

    sig { returns(T.untyped) }
    def call
      @notes = @notebook.not_nil!.notes.published
    end
  end
end
