# typed: strict
# frozen_string_literal: true

module Spaces
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::SidebarSettable

    around_action :set_locale
    before_action :require_authentication
    before_action :set_joined_lists

    sig { returns(T.untyped) }
    def call
      @last_note_modified_lists = viewer!.last_note_modified_lists.limit(10)
      @last_modified_notes = viewer!.last_modified_notes.limit(31)
    end
  end
end
