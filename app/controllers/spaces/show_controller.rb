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
      @last_page_modified_topics = T.let(viewer!.last_page_modified_topics.limit(10), T.nilable(Topic::PrivateRelation))
      @last_modified_pages = T.let(viewer!.last_modified_pages.limit(31), T.nilable(Page::PrivateRelation))
    end
  end
end
