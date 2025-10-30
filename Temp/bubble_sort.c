#include <stdio.h>
int main() {
    int arr[100],n,i,j,temp;
    printf("Enter how many numbers you want to sort: ");
    scanf("%d", &n);
    printf("Enter %d numbers:\n", n);
    for (i = 0; i < n; i++) {
        scanf("%d", &arr[i]);
    }
    for (i = 0; i < n - 1; i++) {             
        for (j = 0; j < n - i - 1; j++) {     
            if (arr[j] < arr[j + 1]) {        
                temp = arr[j];
                arr[j] = arr[j + 1];
                arr[j + 1] = temp;
            }
        }
    }
    printf("\nNumbers in descending order are:\n");
    for (i = 0; i < n; i++) {
        printf("%d ", arr[i]);
    }
     printf("\n");
    return 0;
}

