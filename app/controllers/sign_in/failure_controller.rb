# typed: strict
# frozen_string_literal: true

class SignIn::FailureController < ApplicationController
  extend T::Sig

  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
  end
end
