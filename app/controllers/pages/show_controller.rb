# typed: true
# frozen_string_literal: true

module Pages
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::PageSettable

    around_action :set_locale
    before_action :require_authentication
    before_action :set_page

    sig { returns(T.untyped) }
    def call
      authorize(@page, :show?)

      @link_list = T.let(@page.not_nil!.fetch_link_list, T.nilable(LinkList))
      @backlink_list = T.let(@page.not_nil!.fetch_backlink_list, T.nilable(BacklinkList))
    end
  end
end
