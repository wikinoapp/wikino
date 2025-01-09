# typed: true
# frozen_string_literal: true

module TrashedPages
  class CreateController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::Authorizable

    around_action :set_locale
    before_action :set_current_space
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      @page = Current.space!.find_pages_by_number!(params[:page_number]&.to_i)

      MovePageToTrashUseCase.new.call(page: @page.not_nil!)

      flash[:notice] = t("messages.page.moved_to_trash")
      redirect_to topic_path(Current.space!.identifier, @page.topic.number)
    end
  end
end
