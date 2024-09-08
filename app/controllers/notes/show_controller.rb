# typed: true
# frozen_string_literal: true

module Notes
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SidebarResourcesSettable
    include ControllerConcerns::NoteSettable

    around_action :set_locale
    before_action :require_authentication
    before_action :set_joined_lists
    before_action :set_note

    sig { returns(T.untyped) }
    def call
      authorize(@note, :show?)

      @link_list = @note.fetch_link_list
      @backlink_list = @note.fetch_backlink_list
    end
  end
end
