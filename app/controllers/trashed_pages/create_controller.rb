# typed: true
# frozen_string_literal: true

module TrashedPages
  class CreateController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::PageSettable

    around_action :set_locale
    before_action :set_current_space
    before_action :require_authentication
    before_action :set_page

    sig { returns(T.untyped) }
    def call
      MovePageToTrashUseCase.new.call(page: @page.not_nil!)

      flash[:notice] = t("messages.page.moved_to_trash")
      redirect_to topic_path(Current.space!.identifier, @page.topic.number)
    end
  end
end
