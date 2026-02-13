# typed: true
# frozen_string_literal: true

module MissionControl
  class ApplicationController < ActionController::Base
    extend T::Sig

    include ControllerConcerns::Authenticatable

    before_action :restore_user_session
    before_action :require_admin

    sig { void }
    private def require_admin
      if current_user_record&.atname != "shimbaco"
        raise ActionController::RoutingError, "Not Found"
      end
    end
  end
end
