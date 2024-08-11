# typed: strict
# frozen_string_literal: true

class Layouts::BasicLayoutComponent < ApplicationComponent
  sig { params(with_footer: T::Boolean, main_class_name: String).void }
  def initialize(with_footer: true, main_class_name: "")
    @with_footer = with_footer
    @main_class_name = main_class_name
  end

  sig { returns(T::Boolean) }
  attr_reader :with_footer
  private :with_footer

  sig { returns(String) }
  attr_reader :main_class_name
  private :main_class_name
end
