# typed: true
# frozen_string_literal: true

module Lists
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SidebarResourcesSettable
    include ControllerConcerns::ListSettable

    around_action :set_locale
    before_action :require_authentication
    before_action :set_joined_lists
    before_action :set_list

    sig { returns(T.untyped) }
    def call
      @notes = @list.not_nil!.notes.published
    end
  end
end
