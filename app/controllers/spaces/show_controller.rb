# typed: strict
# frozen_string_literal: true

module Spaces
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable

    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      @last_note_modified_notebooks = viewer!.last_note_modified_notebooks.limit(10)
      @last_modified_notes = viewer!.last_modified_notes.limit(31)
    end
  end
end
