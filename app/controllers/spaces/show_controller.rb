# typed: strict
# frozen_string_literal: true

module Spaces
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::Authorizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      @last_note_modified_notebooks = T.let(viewer!.last_note_modified_notebooks.limit(10), T.nilable(Notebook::PrivateRelation))
      @last_modified_notes = T.let(viewer!.last_modified_notes.limit(31), T.nilable(Note::PrivateRelation))
    end
  end
end
