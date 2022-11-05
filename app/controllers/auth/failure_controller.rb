# typed: true
# frozen_string_literal: true

class Auth::FailureController < ApplicationController
  extend T::Sig

  def call
    @error_msg = request.params["message"]
  end
end
