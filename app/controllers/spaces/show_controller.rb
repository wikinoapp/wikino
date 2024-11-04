# typed: strict
# frozen_string_literal: true

module Spaces
  class ShowController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::Authorizable

    around_action :set_locale
    before_action :set_current_space

    sig { returns(T.untyped) }
    def call
      restore_session

      if signed_in?
        @pinned_pages = Current.space!.pages.published.pinned.order(pinned_at: :desc, id: :desc)
        @pages = Current.space!.pages.published.not_pinned.order(modified_at: :desc, id: :desc).limit(30)
      else
        @pinned_pages = Current.space!.pages.published.pinned
          .joins(:topic).merge(Topic.visibility_public)
          .order(pinned_at: :desc, id: :desc)
        @pages = Current.space!.pages.published.not_pinned
          .joins(:topic).merge(Topic.visibility_public)
          .order(modified_at: :desc, id: :desc)
          .limit(30)
      end
    end
  end
end
