# typed: strict
# frozen_string_literal: true

class SignOut::ShowController < ApplicationController
  extend T::Sig

  include Authenticatable

  before_action :authenticate_user

  sig { returns(T.untyped) }
  def call
    redirect_to sign_out_with_auth0_url(return_to: sign_out_callback_url)
  end
end
