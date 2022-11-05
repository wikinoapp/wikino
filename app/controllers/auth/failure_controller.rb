# typed: true
# frozen_string_literal: true

class Auth::FailureController < ActionController::Base
  extend T::Sig

  def call
    @error_msg = request.params["message"]
  end
end
