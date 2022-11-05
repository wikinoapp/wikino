# typed: strong
# frozen_string_literal: true

module Authenticatable
  def self.helper_method(*methods); end
  def flash; end
  def redirect_to(*args); end
  def reset_session; end
  def root_path; end
  def session; end
  def t(*args); end
end
