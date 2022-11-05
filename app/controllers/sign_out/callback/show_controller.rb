# typed: strict
# frozen_string_literal: true

class SignOut::Callback::ShowController < ApplicationController
  extend T::Sig

  include Authenticatable

  before_action :authenticate_user

  sig { returns(T.untyped) }
  def call
    sign_out

    flash[:notice] = t("messages.sign_out.callback.sign_out_success")
    redirect_to root_path
  end
end
