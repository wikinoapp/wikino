# typed: true
# frozen_string_literal: true

module SwitchableAccounts
  class IndexController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::Authorizable

    layout false

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      @users = User
        .joins(:space)
        .kept
        .where(id: cookie_user_ids)
        .where.not(id: Current.user.id)
        .merge(Space.order(:identifier))
        .preload(:space, :sessions)
    end
  end
end
