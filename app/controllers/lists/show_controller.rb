# typed: true
# frozen_string_literal: true

module Lists
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::Authorizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      @list = viewer!.space.lists.kept.find_by!(identifier: params[:list_identifier])
      authorize(@list, :show?)
    end
  end
end
