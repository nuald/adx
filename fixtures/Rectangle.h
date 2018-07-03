// Class: Rectangle
// Rectangle class
class Rectangle {
    int width, height;
  public:

    // Method: set_values
    // Set width and height of the rectangle.
    //
    // @param width The width.
    // @param height The height.
    void set_values(int width, int height);

    // Method: area
    // Calculates the rectangle area.
    //
    // @return The area of the rectangle.
    int area() {return width*height;}
};
